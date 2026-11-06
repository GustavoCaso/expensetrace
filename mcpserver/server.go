// Package mcpserver exposes a read-only Model Context Protocol (MCP) endpoint
// so AI agents can query a user's expenses, categories, reports and budgets.
//
// Clients authenticate with a per-user API token sent as a bearer token. The
// token determines whose data every tool call reads, so tools never accept a
// user ID as input.
package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GustavoCaso/expensetrace/logger"
	"github.com/GustavoCaso/expensetrace/service/apitoken"
	"github.com/GustavoCaso/expensetrace/service/category"
	"github.com/GustavoCaso/expensetrace/service/expense"
	"github.com/GustavoCaso/expensetrace/service/report"
)

// Path is where the MCP endpoint is mounted.
const Path = "/mcp"

const (
	serverName    = "expensetrace"
	serverVersion = "1.0.0"
	tokenIDKey    = "token_id"
)

const instructions = `ExpenseTrace holds one user's personal transactions, categories and budgets.
All tools are read-only.

Conventions:
- Money is returned in major currency units (e.g. 12.50), alongside a currency code where relevant.
- In expense listings, amounts are signed: charges are negative, income is positive.
- In reports and budgets, spending figures are positive totals.
- Dates use YYYY-MM-DD. Months are 1-12.
- Transactions in the "🚫 Exclude" category are left out of reports and budgets.

Start with get_monthly_report or get_spending_trend for an overview, then use list_expenses to drill down.`

// Services are the business services the MCP tools read from.
type Services struct {
	Expense  *expense.Service
	Category *category.Service
	Report   *report.Service
	Tokens   *apitoken.Service
}

type server struct {
	Services

	logger *logger.Logger
	now    func() time.Time
}

// NewHandler returns the HTTP handler serving the MCP endpoint, including
// bearer-token authentication.
func NewHandler(services Services, logger *logger.Logger) http.Handler {
	s := &server{
		Services: services,
		logger:   logger,
		now:      time.Now,
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   "ExpenseTrace",
		Version: serverVersion,
	}, &mcp.ServerOptions{
		Instructions: instructions,
	})

	s.registerTools(mcpServer)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		// Every tool call is a self-contained read, so no session state is
		// needed. Stateless mode also keeps the server horizontally simple.
		Stateless:    true,
		JSONResponse: true,
	})

	return auth.RequireBearerToken(s.verifyToken, &auth.RequireBearerTokenOptions{
		// API tokens do not expire; they are valid until revoked.
		AllowMissingExpiration: true,
	})(handler)
}

func (s *server) verifyToken(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	apiToken, err := s.Tokens.Authenticate(ctx, token)
	if err != nil {
		if errors.Is(err, apitoken.ErrInvalidToken) {
			return nil, auth.ErrInvalidToken
		}
		s.logger.Error("Failed to verify MCP bearer token", "error", err)
		return nil, err
	}

	return &auth.TokenInfo{
		UserID: strconv.FormatInt(apiToken.UserID(), 10),
		Extra:  map[string]any{tokenIDKey: apiToken.ID()},
	}, nil
}

// errUnauthenticated is returned if a tool somehow runs without a verified
// token. RequireBearerToken should make this unreachable.
var errUnauthenticated = errors.New("unauthenticated")

// caller extracts the authenticated user and token IDs from a request.
func caller(req *mcp.CallToolRequest) (int64, int64, error) {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
		return 0, 0, errUnauthenticated
	}

	userID, err := strconv.ParseInt(req.Extra.TokenInfo.UserID, 10, 64)
	if err != nil {
		return 0, 0, errUnauthenticated
	}

	tokenID, _ := req.Extra.TokenInfo.Extra[tokenIDKey].(int64)

	return userID, tokenID, nil
}

// toolHandler is the shape every tool implements: it receives the
// authenticated user ID and the decoded input.
type toolHandler[In, Out any] func(ctx context.Context, userID int64, in In) (Out, error)

// userError is an error whose message is safe and useful to show the agent,
// such as an invalid argument.
type userError struct {
	msg string
}

func (e *userError) Error() string {
	return e.msg
}

func invalidArg(msg string) error {
	return &userError{msg: msg}
}

// addTool registers a read-only tool, wrapping the handler with
// authentication and an audit log line per call. Internal errors are logged
// and replaced with a generic message so storage details never reach the
// agent.
func addTool[In, Out any](s *server, mcpServer *mcp.Server, tool *mcp.Tool, h toolHandler[In, Out]) {
	openWorld := false
	tool.Annotations = &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  &openWorld,
	}

	mcp.AddTool(
		mcpServer,
		tool,
		func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
			var zero Out

			userID, tokenID, err := caller(req)
			if err != nil {
				return nil, zero, err
			}

			start := s.now()
			out, err := h(ctx, userID, in)

			s.logger.Info("MCP tool call",
				"tool", tool.Name,
				"user_id", userID,
				"token_id", tokenID,
				"duration", time.Since(start).String(),
				"ok", err == nil,
			)

			if err != nil {
				var ue *userError
				if errors.As(err, &ue) {
					return nil, zero, err
				}
				s.logger.Error("MCP tool failed", "tool", tool.Name, "user_id", userID, "error", err)
				return nil, zero, errors.New("internal error while reading data")
			}

			return nil, out, nil
		},
	)
}
