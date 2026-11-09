/**
 * Financial bar chart with savings line overlay
 *
 * Visualizes income, spending, and savings for the selected report range:
 * - Bar chart for income and spending
 * - Line graph for savings trend
 * - Interactive tooltips; clicking a month loads its report card
 * - Every month in the range fits the available width (no scrolling)
 * - Responsive design with high-DPI support
 */

import { formatMoney } from '../formatters.js';

export function initializeBarChart() {
  // Only initialize if the chart container exists
  if (!document.querySelector('.savings-chart-container')) return;

  const canvas = document.getElementById('finance-chart');
  const ctx = canvas.getContext('2d');
  const tooltip = document.getElementById('chart-tooltip');

  // Get device pixel ratio for high-resolution rendering
  const dpr = window.devicePixelRatio || 1;

  // Get chart data from the server
  let chartData = canvas.getAttribute('data-finance')
  if (!chartData || chartData.length === 0) return;

  // Parse the JSON data
  try {
    chartData = JSON.parse(chartData);
  } catch (error) {
    console.error('Error parsing chart data:', error);
    return;
  }
  if (!Array.isArray(chartData) || chartData.length === 0) return;

  // Chart configuration
  const config = {
    padding: { top: 40, right: 40, bottom: 60, left: 80 },
    maxBarWidth: 30,
    maxBarGap: 15,
    labelFont: "12px 'Spline Sans Mono', monospace",
    colors: {
      income_stroke: '#177245',
      expense_stroke: '#B3402E',
      income_fill: 'rgba(23, 114, 69, 0.45)',
      expense_fill: 'rgba(179, 64, 46, 0.45)',
      savings: '#1D5C56',
      grid: '#DDE2DB',
      text: '#67766F',
      tooltip: 'rgba(22, 33, 31, 0.92)'
    }
  };

  // State management
  let hoverPoint = null;

  // "November 2026" -> "Nov 26"; keeps the axis readable when many months show.
  function shortLabel(month) {
    const [name, year] = month.split(' ');
    if (!name || !year) return month;
    return `${name.slice(0, 3)} ${year.slice(-2)}`;
  }

  // Find maximum values for scaling
  function calculateScales(data) {
    let maxValue = 0;

    data.forEach(point => {
      maxValue = Math.max(maxValue, point.Income, Math.abs(point.Spending));
    });

    // Add 10% margin; avoid a zero scale when the range has no data.
    return (maxValue || 100) * 1.1;
  }

  // Draw function
  function drawChart() {
    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const chartWidth = canvas.width / dpr - config.padding.left - config.padding.right;
    const chartHeight = canvas.height / dpr - config.padding.top - config.padding.bottom;

    // Split the width evenly between months, then size bars to fit their slot.
    const slotWidth = chartWidth / chartData.length;
    const barWidth = Math.min(config.maxBarWidth, slotWidth * 0.32);
    const barGap = Math.min(config.maxBarGap, slotWidth * 0.1);
    const groupWidth = barWidth * 2 + barGap;

    const maxValue = calculateScales(chartData);

    // Calculate scale for the y-axis
    const yScale = chartHeight / maxValue;

    // Draw grid lines and labels
    ctx.beginPath();
    ctx.strokeStyle = config.colors.grid;
    ctx.lineWidth = 1;

    // Horizontal grid lines (5 lines)
    for (let i = 0; i <= 5; i++) {
      const y = config.padding.top + chartHeight - (i * chartHeight / 5);

      ctx.moveTo(config.padding.left, y);
      ctx.lineTo(config.padding.left + chartWidth, y);

      // Value labels
      const value = (i * maxValue / 5) / 100;
      ctx.fillStyle = config.colors.text;
      ctx.font = config.labelFont;
      ctx.textAlign = 'right';
      ctx.fillText(value.toFixed(2) + '€', config.padding.left - 10, y + 4);
    }
    ctx.stroke();

    // Draw x-axis
    ctx.beginPath();
    ctx.strokeStyle = config.colors.text;
    ctx.lineWidth = 2;
    ctx.moveTo(config.padding.left, config.padding.top + chartHeight);
    ctx.lineTo(config.padding.left + chartWidth, config.padding.top + chartHeight);
    ctx.stroke();

    // Show every Nth month label so labels never overlap.
    ctx.font = config.labelFont;
    const labelWidth = ctx.measureText('Mmm 00').width + 8;
    const labelStep = Math.max(1, Math.ceil(labelWidth / slotWidth));

    // Draw month labels and bars
    chartData.forEach((point, index) => {
      const slotCenter = config.padding.left + (index + 0.5) * slotWidth;
      const x = slotCenter - groupWidth / 2;

      // Draw month label
      if (index % labelStep === 0) {
        ctx.fillStyle = config.colors.text;
        ctx.font = config.labelFont;
        ctx.textAlign = 'center';
        ctx.fillText(shortLabel(point.Month), slotCenter, config.padding.top + chartHeight + 25);
      }

      // Calculate bar heights
      const incomeHeight = point.Income * yScale;
      const spendingHeight = Math.abs(point.Spending) * yScale;
      const radius = Math.min(10, barWidth / 2);

      // Draw income bar
      ctx.beginPath();
      ctx.fillStyle = config.colors.income_fill;
      const incomeBar = {
        x: x,
        y: config.padding.top + chartHeight - incomeHeight,
        width: barWidth,
        height: incomeHeight
      };
      ctx.roundRect(incomeBar.x, incomeBar.y, incomeBar.width, incomeBar.height, [radius, radius, 0, 0]);
      ctx.fill();

      // Draw spending bar
      ctx.beginPath();
      ctx.fillStyle = config.colors.expense_fill;
      const spendingBar = {
        x: x + barWidth + barGap,
        y: config.padding.top + chartHeight - spendingHeight,
        width: barWidth,
        height: spendingHeight
      };
      ctx.roundRect(spendingBar.x, spendingBar.y, spendingBar.width, spendingBar.height, [radius, radius, 0, 0]);
      ctx.fill();

      // Store these for hit detection
      point.bars = {
        income: incomeBar,
        spending: spendingBar,
        radius
      };
    });

    // Draw savings line
    ctx.beginPath();
    ctx.strokeStyle = config.colors.savings;
    ctx.lineWidth = 3;
    ctx.lineJoin = 'round';

    chartData.forEach((point, index) => {
      const x = config.padding.left + (index + 0.5) * slotWidth;
      // Calculate y coordinate with bounds checking
      // Ensure savings point stays within the chart area
      const rawY = config.padding.top + chartHeight - (point.Savings * yScale);
      // Clamp the y value to stay within the chart boundaries with a small margin
      const margin = -15; // Pixels from edge to ensure visibility
      const y = Math.min(
        Math.max(rawY, config.padding.top + margin),
        config.padding.top + chartHeight - margin
      );

      if (index === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }

      // Store point position for hit detection
      point.savingsPoint = { x, y, radius: Math.min(6, Math.max(3, slotWidth / 6)) };
    });
    ctx.stroke();

    // Draw savings points
    chartData.forEach(point => {
      const { x, y, radius } = point.savingsPoint;

      ctx.beginPath();
      ctx.arc(x, y, radius, 0, Math.PI * 2);
      ctx.fillStyle = point.Savings >= 0 ? config.colors.income_stroke : config.colors.expense_stroke;
      ctx.fill();
      ctx.strokeStyle = '#FCFCFA';
      ctx.lineWidth = 2;
      ctx.stroke();
    });

    // Draw hover indicator
    if (hoverPoint) {
      const point = hoverPoint.point;
      const type = hoverPoint.type;

      ctx.beginPath();
      if (type === 'savings') {
        const { x, y, radius } = point.savingsPoint;
        ctx.arc(x, y, radius + 3, 0, Math.PI * 2);
        ctx.strokeStyle = point.Savings >= 0 ? config.colors.income_stroke : config.colors.expense_stroke;
        ctx.lineWidth = 2;
      } else {
        const bar = point.bars[type];
        const r = point.bars.radius;
        ctx.strokeStyle = type === 'income' ? config.colors.income_stroke : config.colors.expense_stroke;
        ctx.lineWidth = 2;
        ctx.roundRect(bar.x, bar.y, bar.width, bar.height, [r, r, 0, 0]);
      }
      ctx.stroke();
    }
  }

  // Handle mouse movement for tooltips
  function handleMouseMove(e) {
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    let found = false;

    // Check each data point
    for (const point of chartData) {
      // Check savings point
      const sp = point.savingsPoint;
      const distance = Math.sqrt((x - sp.x) ** 2 + (y - sp.y) ** 2);

      if (distance <= sp.radius + 3) {
        hoverPoint = { point, type: 'savings' };
        found = true;

        // Show tooltip
        tooltip.innerHTML = `<strong>${point.Month}</strong><br>
                                  Savings: ${formatMoney(point.Savings)}<br>
                                  (${point.SavingsPercentage.toFixed(1)}%)`;
        tooltip.style.left = `${e.clientX - rect.left}px`;
        tooltip.style.top = `${e.clientY - rect.top - 70}px`;
        tooltip.classList.add('visible');
        break;
      }

      // Check income bar
      const ib = point.bars.income;
      if (x >= ib.x && x <= ib.x + ib.width &&
        y >= ib.y && y <= ib.y + ib.height) {
        hoverPoint = { point, type: 'income' };
        found = true;

        tooltip.innerHTML = `<strong>${point.Month}</strong><br>
                                  Income: ${formatMoney(point.Income)}`;
        tooltip.style.left = `${e.clientX - rect.left}px`;
        tooltip.style.top = `${e.clientY - rect.top - 60}px`;
        tooltip.classList.add('visible');
        break;
      }

      // Check spending bar
      const sb = point.bars.spending;
      if (x >= sb.x && x <= sb.x + sb.width &&
        y >= sb.y && y <= sb.y + sb.height) {
        hoverPoint = { point, type: 'spending' };
        found = true;

        tooltip.innerHTML = `<strong>${point.Month}</strong><br>
                                  Spending: ${formatMoney(point.Spending)}`;
        tooltip.style.left = `${e.clientX - rect.left}px`;
        tooltip.style.top = `${e.clientY - rect.top - 60}px`;
        tooltip.classList.add('visible');
        break;
      }
    }

    if (!found) {
      hoverPoint = null;
      canvas.style.cursor = 'default'
      tooltip.classList.remove('visible');
    } else {
      canvas.style.cursor = 'pointer'
    }

    // Redraw the chart
    drawChart();
  }

  function hanldeClickEvent() {
    if (hoverPoint) {
      const point = hoverPoint.point;

      // Decode the JSON-escaped URL by parsing and re-stringify without escaping
      const decodedUrl = JSON.parse(`"${point.URL}"`);

      htmx.ajax('GET', decodedUrl, { target: '#report', swap: 'outerHTML' })
    }
  }

  // Handle mouse leave
  function handleMouseLeave() {
    hoverPoint = null;
    canvas.style.cursor = 'default';
    tooltip.classList.remove('visible');
    drawChart();
  }

  // Size the canvas to its container with high-DPI support. The container's
  // CSS height shrinks on small screens, so read it rather than the canvas
  // attribute; otherwise the canvas overflows onto the legend.
  function resizeCanvas(container) {
    const displayWidth = container.clientWidth;
    const displayHeight = container.clientHeight || 400;

    canvas.style.width = displayWidth + 'px';
    canvas.style.height = displayHeight + 'px';
    canvas.width = displayWidth * dpr;
    canvas.height = displayHeight * dpr;

    // Resizing the canvas resets the context transform
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  }

  // Initialize the chart
  function initChart() {
    const container = canvas.parentElement;

    resizeCanvas(container);
    drawChart();

    // Add event listeners
    canvas.addEventListener('mousemove', handleMouseMove);
    canvas.addEventListener('mouseleave', handleMouseLeave);
    canvas.addEventListener('click', hanldeClickEvent);

    // Handle window resize
    window.addEventListener('resize', () => {
      resizeCanvas(container);
      drawChart();
    });
  }

  // Initialize everything
  initChart();
}
