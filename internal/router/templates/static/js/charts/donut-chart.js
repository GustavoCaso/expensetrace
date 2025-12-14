/**
 * Donut chart visualization for category breakdown
 *
 * Features:
 * - Interactive donut chart with hover effects
 * - Budget status indicators
 * - Category details panel
 * - Click-to-select segments
 * - Responsive design with high-DPI support
 */

import { formatMoney, formatDate } from '../formatters.js';

export function initializeDonutChart() {
  let expensesDonutChart = null;
  let incomeDonutChart = null;

  // Function to initialize donut charts
  function initializeDonutCharts() {
    const expensesCanvas = document.getElementById('expensesDonutChart');
    const incomeCanvas = document.getElementById('incomeDonutChart');

    // Check if donut chart containers exist
    if (!expensesCanvas && !incomeCanvas) {
      console.log('Donut chart canvases not found');
      return;
    }

    // Initialize expenses donut chart
    if (expensesCanvas) {
      let expensesData = expensesCanvas.getAttribute('data-categories');
      if (expensesData && expensesData.length > 0) {
        try {
          const expensesParsedData = JSON.parse(expensesData);

          // Clean up old instance if it exists
          if (expensesDonutChart) {
            console.log('Cleaning up old expenses donut chart instance');
            expensesDonutChart.destroy();
          }

          // Create new expenses donut chart instance
          expensesDonutChart = new DonutChart('expensesDonutChart', expensesParsedData);
          console.log('Expenses donut chart initialized successfully');
        } catch (error) {
          console.error('Error parsing expenses chart data:', error);
        }
      }
    }

    // Initialize income donut chart
    if (incomeCanvas) {
      let incomeData = incomeCanvas.getAttribute('data-categories');
      if (incomeData && incomeData.length > 0) {
        try {
          const incomeParsedData = JSON.parse(incomeData);

          // Clean up old instance if it exists
          if (incomeDonutChart) {
            console.log('Cleaning up old income donut chart instance');
            incomeDonutChart.destroy();
          }

          // Create new income donut chart instance
          incomeDonutChart = new DonutChart('incomeDonutChart', incomeParsedData);
          console.log('Income donut chart initialized successfully');
        } catch (error) {
          console.error('Error parsing income chart data:', error);
        }
      }
    }
  }

  initializeDonutCharts();

  // Event delegation for legend clicks and close button
  document.body.addEventListener('click', function (event) {
    // Handle legend item clicks
    const legendItem = event.target.closest('.donut-legend .legend-item');
    if (legendItem) {
      const index = parseInt(legendItem.getAttribute('data-index'), 10);
      if (!isNaN(index)) {
        // Determine which chart based on the parent container
        const isExpenseChart = legendItem.closest('#expensesDonutChart');
        if (isExpenseChart && expensesDonutChart) {
          expensesDonutChart.selectSegment(index);
        } else if (incomeDonutChart) {
          incomeDonutChart.selectSegment(index);
        }
      }
      return;
    }

    // Handle close details button clicks
    const closeButton = event.target.closest('.close-details');
    if (closeButton) {
      // Close details for both charts
      if (expensesDonutChart) {
        expensesDonutChart.hideCategoryDetails();
      }
      if (incomeDonutChart) {
        incomeDonutChart.hideCategoryDetails();
      }
      return;
    }
  });

  // Reinitialize donut chart after HTMX swaps content
  document.body.addEventListener('htmx:afterSwap', function (event) {
    // Check if the swapped element is or contains the report element
    if (event.detail.target.id === 'report' || event.detail.target.closest('#report')) {
      console.log('HTMX swapped report content, reinitializing donut chart');
      // Use setTimeout to ensure DOM is fully updated
      setTimeout(() => {
        initializeDonutCharts();
      }, 50);
    }
  });
}


class DonutChart {
  constructor(canvasId, data) {
    this.canvas = document.getElementById(canvasId);
    if (!this.canvas) {
      console.error('Canvas element not found');
      return;
    }
    this.ctx = this.canvas.getContext('2d');
    this.type = this.canvas.getAttribute('data-type') || 'expense';
    this.data = data || [];
    this.config = {
      padding: 30,
      donutThickness: 60,      // Thickness of the donut ring
      centerRadius: 80,         // Inner radius (hole size)
      budgetIndicatorWidth: 8,  // Width of budget status indicator
      colors: {
        expense_base: '#ef4444',  // Red for expenses
        income_base: '#10b981',   // Green for income
        budget_under: '#22c55e',  // Green border for under budget
        budget_near: '#eab308',   // Yellow border for near budget
        budget_over: '#dc2626',   // Red border for over budget
        hover_alpha: 0.7
      }
    };
    this.hoveredSegment = null;
    this.selectedSegment = null;
    this.segments = [];

    this.init();
  }

  init() {
    if (this.data.length === 0) {
      console.warn('No category data available for donut chart');
      return;
    }
    this.setupCanvas();
    this.processData();
    this.setupEventListeners();
    this.draw();
  }

  setupCanvas() {
    // High-DPI display support (similar to existing chart)
    const rect = this.canvas.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    this.canvas.width = rect.width * dpr;
    this.canvas.height = rect.height * dpr;
    this.ctx.scale(dpr, dpr);
    this.canvas.style.width = rect.width + 'px';
    this.canvas.style.height = rect.height + 'px';
  }

  processData() {
    // Data is already filtered on the server side
    const isExpense = this.type === 'expense';

    if (this.data.length === 0) {
      console.warn(`No ${this.type} data available for donut chart`);
      return;
    }

    // Calculate total for percentages
    const total = this.data.reduce((sum, cat) => {
      return sum + Math.abs(cat.amount);
    }, 0);

    if (total === 0) {
      console.warn('Total amount is zero, cannot create donut chart');
      return;
    }

    // Generate color palette and calculate angles
    let currentAngle = -Math.PI / 2; // Start at top
    this.segments = [];

    this.data.forEach((category, index) => {
      const percentage = (Math.abs(category.amount) / total) * 100;
      const angleSize = (percentage / 100) * 2 * Math.PI;

      // Generate color based on type
      const baseColor = isExpense
        ? this.generateExpenseColor(index)
        : this.generateIncomeColor(index);

      const segment = {
        category: category,
        color: baseColor,
        startAngle: currentAngle,
        endAngle: currentAngle + angleSize,
        percentage: percentage,
        isExpense: isExpense
      };

      currentAngle += angleSize;
      this.segments.push(segment);
    });
  }

  generateExpenseColor(index) {
    // Predefined palette of distinct warm colors for better differentiation
    // Includes reds, oranges, pinks, burgundy, coral, salmon
    const baseHues = [
      0,    // Red
      15,   // Red-Orange
      30,   // Orange
      45,   // Yellow-Orange
      340,  // Pink-Red
      355,  // Deep Pink
      10,   // Vermillion
      25,   // Coral
      320,  // Magenta-Pink
      5,    // Crimson
      35,   // Amber
      330,  // Rose
    ];

    // Cycle through base hues
    const baseHue = baseHues[index % baseHues.length];

    // Add variation for categories beyond the base palette
    const hueVariation = Math.floor(index / baseHues.length) * 8;
    const hue = (baseHue + hueVariation) % 360;

    // Vary saturation and lightness for additional distinction
    const saturationBase = 65 + (index % 3) * 10;
    const lightnessBase = 45 + (index % 4) * 8;

    return `hsl(${hue}, ${saturationBase}%, ${lightnessBase}%)`;
  }

  generateIncomeColor(index) {
    // Generate shades of green/teal for income
    const hue = 140 + (index * 15) % 40; // Green to teal range (140-180)
    const saturation = 60 + (index * 10) % 30;
    const lightness = 40 + (index * 5) % 20;
    return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
  }

  draw() {
    const rect = this.canvas.getBoundingClientRect();
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;
    const outerRadius = Math.min(centerX, centerY) - this.config.padding;
    const innerRadius = this.config.centerRadius;

    // Clear canvas
    this.ctx.clearRect(0, 0, rect.width, rect.height);

    // Draw each segment
    this.segments.forEach((segment, index) => {
      // Draw main donut segment
      this.ctx.beginPath();
      this.ctx.arc(centerX, centerY, outerRadius, segment.startAngle, segment.endAngle);
      this.ctx.arc(centerX, centerY, innerRadius, segment.endAngle, segment.startAngle, true);
      this.ctx.closePath();

      // Apply color with hover effect
      if (this.hoveredSegment === index) {
        this.ctx.globalAlpha = this.config.colors.hover_alpha;
      }
      this.ctx.fillStyle = segment.color;
      this.ctx.fill();
      this.ctx.globalAlpha = 1.0;

      // Highlight selected segment
      if (this.selectedSegment === index) {
        this.ctx.strokeStyle = '#ffffff';
        this.ctx.lineWidth = 3;
        this.ctx.stroke();
      }
    });

    // Draw center circle (creates the donut hole)
    this.ctx.beginPath();
    this.ctx.arc(centerX, centerY, innerRadius, 0, 2 * Math.PI);
    this.ctx.fillStyle = '#ffffff';
    this.ctx.fill();

    // Draw total in center
    this.drawCenterText(centerX, centerY);
  }

  drawCenterText(centerX, centerY) {
    const total = this.segments.reduce((sum, s) => sum + Math.abs(s.category.amount), 0);
    const isExpense = this.type === 'expense';

    this.ctx.textAlign = 'center';
    this.ctx.textBaseline = 'middle';

    // Draw label
    this.ctx.font = 'bold 16px sans-serif';
    this.ctx.fillStyle = '#374151';
    this.ctx.fillText('Total', centerX, centerY - 15);

    // Draw total amount with appropriate color
    this.ctx.font = 'bold 24px sans-serif';
    this.ctx.fillStyle = isExpense
      ? this.config.colors.expense_base
      : this.config.colors.income_base;
    this.ctx.fillText(formatMoney(total), centerX, centerY + 15);
  }

  setupEventListeners() {
    // Bind event handlers to maintain 'this' context
    this.boundHandleMouseMove = (e) => this.handleMouseMove(e);
    this.boundHandleClick = (e) => this.handleClick(e);
    this.boundHandleMouseLeave = () => {
      this.hoveredSegment = null;
      this.draw();
      this.hideTooltip();
    };
    this.boundHandleResize = () => {
      this.setupCanvas();
      this.draw();
    };

    // Mouse move for hover effects and tooltip
    this.canvas.addEventListener('mousemove', this.boundHandleMouseMove);

    // Click for selecting segment and showing details
    this.canvas.addEventListener('click', this.boundHandleClick);

    // Mouse leave to clear hover state
    this.canvas.addEventListener('mouseleave', this.boundHandleMouseLeave);

    // Window resize
    window.addEventListener('resize', this.boundHandleResize);
  }

  handleMouseMove(e) {
    const rect = this.canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;

    const segment = this.getSegmentAtPoint(x, y, centerX, centerY);

    if (segment !== this.hoveredSegment) {
      this.hoveredSegment = segment;
      this.draw();

      if (segment !== null) {
        this.showTooltip(e, this.segments[segment]);
        this.canvas.style.cursor = 'pointer';
      } else {
        this.hideTooltip();
        this.canvas.style.cursor = 'default';
      }
    } else if (segment !== null) {
      this.updateTooltipPosition(e);
    }
  }

  handleClick(e) {
    const rect = this.canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;

    const segment = this.getSegmentAtPoint(x, y, centerX, centerY);

    if (segment !== null) {
      if (this.selectedSegment === segment) {
        // Deselect if clicking the same segment
        this.selectedSegment = null;
        this.hideCategoryDetails();
      } else {
        // Select new segment
        this.selectedSegment = segment;
        this.showCategoryDetails(this.segments[segment]);
      }
      this.draw();
    }
  }

  getSegmentAtPoint(x, y, centerX, centerY) {
    const dx = x - centerX;
    const dy = y - centerY;
    const distance = Math.sqrt(dx * dx + dy * dy);
    const outerRadius = Math.min(centerX, centerY) - this.config.padding;
    const innerRadius = this.config.centerRadius;

    // Check if point is within donut ring
    if (distance < innerRadius || distance > outerRadius) {
      return null;
    }

    // Calculate angle
    let angle = Math.atan2(dy, dx);
    // Normalize to start from top (-π/2)
    angle = angle + Math.PI / 2;
    if (angle < 0) angle += 2 * Math.PI;

    // Find which segment this angle belongs to
    for (let i = 0; i < this.segments.length; i++) {
      let start = this.segments[i].startAngle + Math.PI / 2;
      let end = this.segments[i].endAngle + Math.PI / 2;

      // Normalize angles
      if (start < 0) start += 2 * Math.PI;
      if (end < 0) end += 2 * Math.PI;
      if (start > 2 * Math.PI) start -= 2 * Math.PI;
      if (end > 2 * Math.PI) end -= 2 * Math.PI;

      if (start <= end) {
        if (angle >= start && angle <= end) return i;
      } else {
        // Handle wrap-around case
        if (angle >= start || angle <= end) return i;
      }
    }

    return null;
  }

  showTooltip(e, segment) {
    let tooltip = document.getElementById('donutTooltip');
    if (!tooltip) {
      tooltip = document.createElement('div');
      tooltip.id = 'donutTooltip';
      tooltip.className = 'donut-tooltip';
      document.body.appendChild(tooltip);
    }

    const budgetInfo = segment.category.budget.status != 'no_budget'
      ? `<div class="budget-status ${segment.category.budget.status}">
           Budget: ${formatMoney(segment.category.budget.amount)}
           (${segment.category.budget.percentage_used.toFixed(0)}% used)
         </div>`
      : '';

    tooltip.innerHTML = `
      <div class="tooltip-color" style="background-color: ${segment.color}"></div>
      <div class="tooltip-content">
        <div class="tooltip-category">${segment.category.name}</div>
        <div class="tooltip-amount">${formatMoney(Math.abs(segment.category.amount))}</div>
        <div class="tooltip-percentage">${segment.percentage.toFixed(1)}% of total</div>
        ${budgetInfo}
      </div>
    `;

    this.updateTooltipPosition(e);
    tooltip.style.display = 'block';
  }

  updateTooltipPosition(e) {
    const tooltip = document.getElementById('donutTooltip');
    if (tooltip) {
      tooltip.style.left = (e.pageX + 10) + 'px';
      tooltip.style.top = (e.pageY + 10) + 'px';
    }
  }

  hideTooltip() {
    const tooltip = document.getElementById('donutTooltip');
    if (tooltip) {
      tooltip.style.display = 'none';
    }
  }

  showCategoryDetails(segment) {
    const detailsContainer = document.getElementById('categoryDetails');
    if (!detailsContainer) return;

    const category = segment.category;

    // Build HTML similar to category.html template
    const html = `
      <div class="category-detail-header">
        <h3>${category.name}</h3>
        <button class="close-details">×</button>
      </div>
      ${this.renderCategoryDetail(category)}
    `;

    detailsContainer.innerHTML = html;
    detailsContainer.style.display = 'block';

    // Scroll to details
    detailsContainer.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }

  hideCategoryDetails() {
    const detailsContainer = document.getElementById('categoryDetails');
    if (detailsContainer) {
      detailsContainer.style.display = 'none';
    }
    this.selectedSegment = null;
    this.draw();
  }

  renderBudgetRemaining(remaining) {
    let formattedRemaining = formatMoney(Math.abs(remaining))
    if (remaining <= 0) {
      return `<div class='budget-warning'>Over budget by ${formattedRemaining}</div>`
    } else {
      return `<div class='budget-remaining'>${formattedRemaining} remaining</div>`
    }
  }

  renderCategoryDetail(category) {
    // Generate HTML matching the structure in category.html
    const budgetSection = category.budget && category.budget.status !== 'no_budget' ? `
      <div class="budget-container">
        <div class="budget-info">
          <span class="budget-spent">${formatMoney(category.budget.spent)}</span>
          <span class="budget-separator">/</span>
          <span class="budget-total">${formatMoney(category.budget.amount)}</span>
        </div>
        <div class="budget-bar">
          <div class="budget-progress budget-${category.budget.status}"
               style="width: ${Math.min(category.budget.percentage_used, 100)}%">
          </div>
        </div>
        ${this.renderBudgetRemaining(category.budget.remaining)}
        </div >` : '';

    const expensesHtml = category.expenses.map(expense => `
      <a href="/expense/${expense.id}" class="table-row" >
        <div class="table-cell">${formatDate(expense.date)}</div>
        <div class="table-cell">${expense.description}</div>
        <div class="table-cell source-column">${expense.source}</div>
        <div class="table-cell ${expense.expense_type === 0 ? 'expense' : 'income'}">
          ${formatMoney(Math.abs(expense.amount))}
        </div>
      </a>
  `).join('');

    return `
      ${budgetSection}
      <div class="category-stats">
        <span>Total: ${formatMoney(Math.abs(category.amount))}</span>
        <span>Transactions: ${category.expenses.length}</span>
        <span>Avg: ${formatMoney(Math.abs(category.average_amount))}</span>
      </div>
      <div class="table">
        <div class="table-header">
          <div class="table-header-cell">Date</div>
          <div class="table-header-cell">Description</div>
          <div class="table-header-cell">Source</div>
          <div class="table-header-cell">Amount</div>
        </div>
        <div class="table-body">
          ${expensesHtml}
        </div>
      </div>
`;
  }

  selectSegment(index) {
    if (this.selectedSegment === index) {
      this.selectedSegment = null;
      this.hideCategoryDetails();
    } else {
      this.selectedSegment = index;
      this.showCategoryDetails(this.segments[index]);
    }
    this.draw();
  }

  destroy() {
    // Remove event listeners
    if (this.canvas) {
      this.canvas.removeEventListener('mousemove', this.boundHandleMouseMove);
      this.canvas.removeEventListener('click', this.boundHandleClick);
      this.canvas.removeEventListener('mouseleave', this.boundHandleMouseLeave);
    }

    // Remove window resize listener
    if (this.boundHandleResize) {
      window.removeEventListener('resize', this.boundHandleResize);
    }

    // Remove tooltip if it exists
    const tooltip = document.getElementById('donutTooltip');
    if (tooltip) {
      tooltip.remove();
    }

    // Clear references
    this.canvas = null;
    this.ctx = null;
    this.data = null;
    this.segments = [];
    this.boundHandleMouseMove = null;
    this.boundHandleClick = null;
    this.boundHandleMouseLeave = null;
    this.boundHandleResize = null;
  }
}
