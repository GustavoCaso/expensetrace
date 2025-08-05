document.addEventListener('DOMContentLoaded', function () {
  // Set up year toggles
  const yearHeaders = document.querySelectorAll('.year-header');
  yearHeaders.forEach(header => {
    header.addEventListener('click', function () {
      const content = this.nextElementSibling;
      const isCollapsed = content.classList.contains('collapsed');

      // Toggle collapse class
      if (isCollapsed) {
        content.classList.remove('collapsed');
        this.classList.remove('collapsed');
      } else {
        content.classList.add('collapsed');
        this.classList.add('collapsed');
      }
    });
  });

  // Set up month toggles
  const monthHeaders = document.querySelectorAll('.month-header');
  monthHeaders.forEach(header => {
    header.addEventListener('click', function (e) {
      // Prevent the click from bubbling up to parent elements
      e.stopPropagation();
      const content = this.nextElementSibling;
      const isCollapsed = content.classList.contains('collapsed');

      // Toggle collapse class
      if (isCollapsed) {
        content.classList.remove('collapsed');
        this.classList.remove('collapsed');
      } else {
        content.classList.add('collapsed');
        this.classList.add('collapsed');
      }
    });
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
      income: '#10b981',
      expense: '#ef4444',
      savings: '#3b82f6',
      grid: '#e5e7eb',
      text: '#6b7280',
      tooltip: 'rgba(31, 41, 55, 0.9)'
    },
    animationDuration: 500,
    visibleMonths: 8 // Number of months visible by default
  };

  // Derived values
  const monthWidth = (config.barWidth * 2) + (config.barGap * 3);

  // State management
  let currentOffset = 0; // Index of first visible month
  let animating = false;
  let hoverPoint = null;

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
      ctx.fillStyle = config.colors.income;
      const incomeBar = {
        x: x,
        y: config.padding.top + chartHeight - incomeHeight,
        width: config.barWidth,
        height: incomeHeight
      };
      ctx.fillRect(incomeBar.x, incomeBar.y, incomeBar.width, incomeBar.height);

      // Draw spending bar
      ctx.fillStyle = config.colors.expense;
      const spendingBar = {
        x: x + config.barWidth + config.barGap,
        y: config.padding.top + chartHeight - spendingHeight,
        width: config.barWidth,
        height: spendingHeight
      };
      ctx.fillRect(spendingBar.x, spendingBar.y, spendingBar.width, spendingBar.height);

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
      ctx.fillStyle = point.Savings >= 0 ? config.colors.income : config.colors.expense;
      ctx.fill();
      ctx.strokeStyle = 'white';
      ctx.lineWidth = 2;
      ctx.stroke();
    });

    // Draw hover indicator
    if (hoverPoint) {
      const point = hoverPoint.point;
      const type = hoverPoint.type;

      if (type === 'savings') {
        const { x, y, radius } = point.savingsPoint;
        ctx.beginPath();
        ctx.arc(x, y, radius + 3, 0, Math.PI * 2);
        ctx.strokeStyle = point.Savings >= 0 ? config.colors.income : config.colors.expense;
        ctx.lineWidth = 2;
        ctx.stroke();
      } else {
        const bar = point.bars[type];
        ctx.strokeStyle = type === 'income' ? config.colors.income : config.colors.expense;
        ctx.lineWidth = 2;
        ctx.strokeRect(bar.x - 2, bar.y - 2, bar.width + 4, bar.height + 4);
      }
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
