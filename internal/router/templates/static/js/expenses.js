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
});
