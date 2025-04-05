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
