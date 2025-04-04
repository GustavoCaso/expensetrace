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
        content.style.maxHeight = content.scrollHeight + 'px';
        this.classList.remove('collapsed');
      } else {
        content.classList.add('collapsed');
        content.style.maxHeight = '0px';
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
      console.log(this)
      const content = this.nextElementSibling;
      console.log(content)
      const isCollapsed = content.classList.contains('collapsed');

      // Toggle collapse class
      if (isCollapsed) {
        content.classList.remove('collapsed');
        content.style.maxHeight = content.scrollHeight + 'px';
        this.classList.remove('collapsed');
      } else {
        content.classList.add('collapsed');
        content.style.maxHeight = '0px';
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
});


document.addEventListener('DOMContentLoaded', function () {
  // Form validation for the new category
  const categoryNameInput = document.getElementById('category-name');
  const categoryPatternInput = document.getElementById('category-pattern');

  if (categoryNameInput) {
    categoryNameInput.addEventListener('input', function () {
      validateCategoryName(this.value);
    });
  }

  if (categoryPatternInput) {
    categoryPatternInput.addEventListener('input', function () {
      validateCategoryPattern(this.value);
    });
  }
});

// Toggle batch action buttons based on selection state
function toggleBatchButtons(enabled) {
  const batchButtons = document.querySelectorAll('#batch-categorize-form button, .batch-actions button');
  batchButtons.forEach(btn => {
    btn.disabled = !enabled;
  });
}

// Validate category name
function validateCategoryName(name) {
  const feedback = document.getElementById('name-validation');
  if (!feedback) return;

  if (name.length < 3) {
    feedback.className = 'validation-feedback validation-error';
    feedback.innerHTML = '<span class="icon">❌</span> Name must be at least 3 characters';
    return false;
  }

  if (name.length > 20) {
    feedback.className = 'validation-feedback validation-error';
    feedback.innerHTML = '<span class="icon">❌</span> Name must be less than 20 characters';
    return false;
  }

  feedback.className = 'validation-feedback validation-success';
  feedback.innerHTML = '<span class="icon">✓</span> Valid name';
  return true;
}

// Validate category pattern
function validateCategoryPattern(pattern) {
  const feedback = document.getElementById('pattern-validation');
  if (!feedback) return;

  if (pattern.length < 3) {
    feedback.className = 'validation-feedback validation-error';
    feedback.innerHTML = '<span class="icon">❌</span> Pattern must be at least 3 characters';
    return false;
  }

  // Check if pattern is a valid regex
  try {
    new RegExp(pattern);
    feedback.className = 'validation-feedback validation-success';
    feedback.innerHTML = '<span class="icon">✓</span> Valid pattern';
    return true;
  } catch (e) {
    feedback.className = 'validation-feedback validation-error';
    feedback.innerHTML = '<span class="icon">❌</span> Invalid regular expression';
    return false;
  }
}
