document.addEventListener('DOMContentLoaded', function () {
  // Set up year and month toggles using event delegation
  document.body.addEventListener('click', function (e) {
    // Handle year header clicks
    if (e.target.closest('.year-header')) {
      const header = e.target.closest('.year-header');
      const content = header.nextElementSibling;
      const isCollapsed = content.classList.contains('collapsed');

      // Toggle collapse class
      if (isCollapsed) {
        content.classList.remove('collapsed');
        header.classList.remove('collapsed');
      } else {
        content.classList.add('collapsed');
        header.classList.add('collapsed');
      }
    }

    // Handle month header clicks
    if (e.target.closest('.month-header')) {
      // Prevent the click from bubbling up to parent elements
      e.stopPropagation();
      const header = e.target.closest('.month-header');
      const content = header.nextElementSibling;
      const isCollapsed = content.classList.contains('collapsed');

      // Toggle collapse class
      if (isCollapsed) {
        content.classList.remove('collapsed');
        header.classList.remove('collapsed');
      } else {
        content.classList.add('collapsed');
        header.classList.add('collapsed');
      }
    }
  });

  // Handle category tabs
  document.body.addEventListener('click', function (e) {
    const tabBtn = e.target.closest('.tab-btn');
    if (tabBtn) {
      e.preventDefault();

      // Get all tab buttons and contents
      const tabBtns = document.querySelectorAll('.tab-btn');
      const tabContents = document.querySelectorAll('.tab-content');

      // Remove active class from all
      tabBtns.forEach(btn => btn.classList.remove('active'));
      tabContents.forEach(content => content.classList.remove('active'));

      // Add active class to clicked tab and its content
      tabBtn.classList.add('active');
      const tabId = tabBtn.getAttribute('data-tab');
      document.getElementById(tabId + '-tab').classList.add('active');
    }
  });

  // Toggle details function
  document.addEventListener('click', function (e) {
    if (e.target.closest('.toggle-details')) {
      const button = e.target.closest('.toggle-details');
      const targetId = button.getAttribute('data-target');
      const targetContent = document.getElementById(targetId);

      if (targetContent) {
        const isCollapsed = targetContent.classList.contains('collapsed');
        const toggleText = button.querySelector('.toggle-text');

        if (isCollapsed) {
          targetContent.classList.remove('collapsed');
          toggleText.textContent = 'Hide Details';
        } else {
          targetContent.classList.add('collapsed');
          toggleText.textContent = 'Show Details';
        }
      }
    }
  });

  // Toggle details function
  document.body.addEventListener('click', function (e) {
    if (e.target.closest('.toggle')) {
      const button = e.target.closest('.toggle');
      const targetId = button.getAttribute('data-target');
      const targetContent = document.getElementById(targetId);

      if (targetContent) {
        const isCollapsed = targetContent.classList.contains('show');

        if (isCollapsed) {
          targetContent.classList.remove('show');
        } else {
          targetContent.classList.add('show');
        }
      }
    }
  });

  // Toggle details swap function
  document.addEventListener('click', function (e) {
    if (e.target.closest('.toggle-details-swap')) {
      const button = e.target.closest('.toggle-details-swap');
      const targetId = button.getAttribute('data-target');
      const contentId = button.getAttribute('data-content');
      const targetContent = document.getElementById(targetId);
      const contentContent = document.getElementById(contentId);

      if (targetContent && contentContent) {
        const isCollapsed = targetContent.classList.contains('collapsed');

        if (isCollapsed) {
          targetContent.classList.remove('collapsed');
          contentContent.classList.add('collapsed')
        } else {
          targetContent.classList.add('collapsed');
          contentContent.classList.remove('collapsed')
        }
      }
    }
  });

  document.addEventListener('click', function (event) {
    // Handle toggle pattern help button
    if (event.target.id === 'toggle-pattern-help' || event.target.closest('#toggle-pattern-help')) {
      const patternHelp = document.getElementById('pattern-help');
      const toggleBtn = document.getElementById('toggle-pattern-help');

      if (patternHelp && toggleBtn) {
        patternHelp.classList.toggle('hidden');
        toggleBtn.textContent = patternHelp.classList.contains('hidden')
          ? 'Need help with patterns?'
          : 'Hide pattern help';
      }
    }
  });

  // Setup functionality when new content is added via HTMX
  document.body.addEventListener('htmx:afterSwap', function (event) {
    // If our category form was just loaded, make sure the pattern help is hidden by default
    const patternHelp = document.getElementById('pattern-help');
    const toggleBtn = document.getElementById('toggle-pattern-help');

    if (patternHelp && toggleBtn) {
      patternHelp.classList.add('hidden');
      toggleBtn.textContent = 'Need help with patterns?'
    }
  });

  // Toggle transactions display in uncategorized expenses
  document.addEventListener('click', function (event) {
    const toggleBtn = event.target.closest('.toggle-transactions');
    if (toggleBtn) {
      const targetId = toggleBtn.getAttribute('data-target');
      const transactionsList = document.getElementById(targetId);

      if (transactionsList) {
        const isCollapsed = transactionsList.classList.contains('collapsed');

        if (isCollapsed) {
          transactionsList.classList.remove('collapsed');
          toggleBtn.textContent = 'Hide transactions';
        } else {
          transactionsList.classList.add('collapsed');
          const count = transactionsList.children.length;
          toggleBtn.textContent = `Show ${count} more transactions`;
        }
      }
    }
  });
});

// Format currency function
function formatMoney(amount) {
  const absAmount = Math.abs(amount);
  const sign = amount < 0 ? '-' : '';

  let intPart = Math.floor(absAmount / 100).toString();
  const decPart = (absAmount % 100).toString().padStart(2, '0');

  // Add thousands separator
  if (intPart.length > 3) {
    intPart = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
  }

  return `${sign}${intPart},${decPart}€`;
}

// Initialize bar chart
document.addEventListener('DOMContentLoaded', function () {
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


  // Chart configuration
  const config = {
    padding: { top: 40, right: 40, bottom: 60, left: 80 },
    barWidth: 30,
    barGap: 15,
    colors: {
      income_stroke: '#10b981',
      expense_stroke: '#ef4444',
      income_fill: 'rgba(16, 185, 129, 0.5)',
      expense_fill: 'rgba(239, 68, 68, 0.5)',
      savings: '#3b82f6',
      grid: '#e5e7eb',
      text: '#6b7280',
      tooltip: 'rgba(31, 41, 55, 0.9)'
    },
    animationDuration: 500,
    visibleMonths: 7 // Number of months visible by default
  };

  // Derived values
  const monthWidth = (config.barWidth * 2) + (config.barGap * 3);

  // State management
  let currentOffset = 0; // Index of first visible month
  let animating = false;
  let hoverPoint = null;

  // Find maximum values for scaling
  function calculateScales(visibleData) {
    let maxIncome = 0;
    let maxSpending = 0;

    visibleData.forEach(point => {
      maxIncome = Math.max(maxIncome, point.Income);
      maxSpending = Math.max(maxSpending, Math.abs(point.Spending));
    });

    const maxValue = Math.max(maxIncome, maxSpending) * 1.1; // Add 10% margin
    return maxValue;
  }

  // Draw function
  function drawChart() {
    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const chartWidth = canvas.width / dpr - config.padding.left - config.padding.right;
    const chartHeight = canvas.height / dpr - config.padding.top - config.padding.bottom;

    // Determine visible data
    const visibleData = chartData.slice(
      currentOffset,
      Math.min(currentOffset + config.visibleMonths, chartData.length)
    );

    const maxValue = calculateScales(visibleData);

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
      ctx.font = '12px Inter, sans-serif';
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

    // Draw month labels and bars
    visibleData.forEach((point, index) => {
      const x = config.padding.left + (index * monthWidth) + config.barGap;

      // Draw month label
      ctx.fillStyle = config.colors.text;
      ctx.font = '12px Inter, sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText(point.Month, x + config.barWidth, config.padding.top + chartHeight + 25);

      // Calculate bar heights
      const incomeHeight = point.Income * yScale;
      const spendingHeight = Math.abs(point.Spending) * yScale;

      // Draw income bar
      ctx.beginPath();
      ctx.fillStyle = config.colors.income_fill;
      const incomeBar = {
        x: x,
        y: config.padding.top + chartHeight - incomeHeight,
        width: config.barWidth,
        height: incomeHeight
      };
      ctx.roundRect(incomeBar.x, incomeBar.y, incomeBar.width, incomeBar.height, [10, 10, 0, 0]);
      ctx.fill();

      // Draw spending bar
      ctx.beginPath();
      ctx.fillStyle = config.colors.expense_fill;
      const spendingBar = {
        x: x + config.barWidth + config.barGap,
        y: config.padding.top + chartHeight - spendingHeight,
        width: config.barWidth,
        height: spendingHeight
      };
      ctx.roundRect(spendingBar.x, spendingBar.y, spendingBar.width, spendingBar.height, [10, 10, 0, 0]);
      ctx.fill();

      // Store these for hit detection
      point.bars = {
        income: incomeBar,
        spending: spendingBar
      };
    });

    // Draw savings line
    ctx.beginPath();
    ctx.strokeStyle = config.colors.savings;
    ctx.lineWidth = 3;
    ctx.lineJoin = 'round';

    visibleData.forEach((point, index) => {
      const x = config.padding.left + (index * monthWidth) + config.barWidth + config.barGap;
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
      point.savingsPoint = { x, y, radius: 6 };
    });
    ctx.stroke();

    // Draw savings points
    visibleData.forEach(point => {
      const { x, y, radius } = point.savingsPoint;

      ctx.beginPath();
      ctx.arc(x, y, radius, 0, Math.PI * 2);
      ctx.fillStyle = point.Savings >= 0 ? config.colors.income_stroke : config.colors.expense_stroke;
      ctx.fill();
      ctx.strokeStyle = 'white';
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
        ctx.strokeStyle = point.Savings >= 0 ? config.colors.income : config.colors.expense;
        ctx.lineWidth = 2;
      } else {
        const bar = point.bars[type];
        ctx.strokeStyle = type === 'income' ? config.colors.income_stroke : config.colors.expense_stroke;
        ctx.lineWidth = 2;
        ctx.roundRect(bar.x, bar.y, bar.width, bar.height, [10, 10, 0, 0]);
      }
      ctx.stroke();
    }

    // Update period display
    const periodText = visibleData.length > 0 ?
      `${visibleData[0].Month} - ${visibleData[visibleData.length - 1].Month}` : '';
    document.getElementById('current-period').textContent = periodText;

    // Update navigation buttons
    document.getElementById('prev-period').disabled = currentOffset <= 0;
    document.getElementById('next-period').disabled =
      currentOffset + config.visibleMonths >= chartData.length;
  }

  // Handle mouse movement for tooltips
  function handleMouseMove(e) {
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Get visible data
    const visibleData = chartData.slice(
      currentOffset,
      Math.min(currentOffset + config.visibleMonths, chartData.length)
    );

    let found = false;

    // Check each data point
    for (const point of visibleData) {
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

  function hanldeClickEvent(e) {
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

  // Handle navigation
  function navigatePrevious() {
    if (currentOffset > 0 && !animating) {
      animating = true;

      // Animate the transition
      const startOffset = currentOffset;
      const targetOffset = Math.max(0, currentOffset - config.visibleMonths);
      const startTime = performance.now();

      function animate(time) {
        const elapsed = time - startTime;
        const progress = Math.min(1, elapsed / config.animationDuration);

        // Easing function for smoother animation
        const easeProgress = 1 - Math.pow(1 - progress, 3);

        currentOffset = startOffset - easeProgress * (startOffset - targetOffset);
        drawChart();

        if (progress < 1) {
          requestAnimationFrame(animate);
        } else {
          currentOffset = targetOffset;
          animating = false;
          drawChart();
        }
      }

      requestAnimationFrame(animate);
    }
  }

  function navigateNext() {
    if (currentOffset + config.visibleMonths < chartData.length && !animating) {
      animating = true;

      // Animate the transition
      const startOffset = currentOffset;
      const targetOffset = Math.min(
        chartData.length - config.visibleMonths,
        currentOffset + config.visibleMonths
      );
      const startTime = performance.now();

      function animate(time) {
        const elapsed = time - startTime;
        const progress = Math.min(1, elapsed / config.animationDuration);

        // Easing function for smoother animation
        const easeProgress = 1 - Math.pow(1 - progress, 3);

        currentOffset = startOffset + easeProgress * (targetOffset - startOffset);
        drawChart();

        if (progress < 1) {
          requestAnimationFrame(animate);
        } else {
          currentOffset = targetOffset;
          animating = false;
          drawChart();
        }
      }

      requestAnimationFrame(animate);
    }
  }

  // Initialize the chart
  function initChart() {
    // Set initial offset to show the most recent months
    if (chartData.length > config.visibleMonths) {
      currentOffset = chartData.length - config.visibleMonths;
    }

    // Set canvas dimensions based on container with high-DPI support
    const container = canvas.parentElement;
    const displayWidth = container.clientWidth;
    const displayHeight = canvas.height;

    // Set the canvas size in CSS pixels
    canvas.style.width = displayWidth + 'px';
    canvas.style.height = displayHeight + 'px';

    // Set actual size in memory (scaled for DPI)
    canvas.width = displayWidth * dpr;
    canvas.height = displayHeight * dpr;

    // Set the most appropiate number of visible months
    config.visibleMonths = Math.floor((canvas.width / dpr - config.padding.left - config.padding.right) / monthWidth);

    // Scale the context to ensure correct drawing operations
    ctx.scale(dpr, dpr);

    // Initial draw
    drawChart();

    // Add event listeners
    canvas.addEventListener('mousemove', handleMouseMove);
    canvas.addEventListener('mouseleave', handleMouseLeave);
    canvas.addEventListener('click', hanldeClickEvent);

    document.getElementById('prev-period').addEventListener('click', navigatePrevious);
    document.getElementById('next-period').addEventListener('click', navigateNext);

    // Handle window resize
    window.addEventListener('resize', () => {
      // Update canvas dimensions with high-DPI support on resize
      const displayWidth = container.clientWidth;

      canvas.style.width = displayWidth + 'px';
      canvas.width = displayWidth * dpr;

      config.visibleMonths = Math.floor((canvas.width / dpr - config.padding.left - config.padding.right) / monthWidth);

      // Reset the scale
      ctx.scale(dpr, dpr);

      drawChart();
    });
  }

  // Initialize everything
  initChart();
});


// Initialize donut chart
document.addEventListener('DOMContentLoaded', function () {
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
});


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

function formatDate(dateString) {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}
