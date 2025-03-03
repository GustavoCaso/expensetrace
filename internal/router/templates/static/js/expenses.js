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

  // Calculate and display month totals
  // calculateMonthTotals();

  // Add search functionality
  // const searchInput = document.getElementById('expense-search');
  // if (searchInput) {
  //   searchInput.addEventListener('input', function () {
  //     filterExpenses(this.value.toLowerCase());
  //   });
  // }
});

// function calculateMonthTotals() {
//   const months = document.querySelectorAll('.expense-month');

//   months.forEach(month => {
//     let totalIncome = 0;
//     let totalExpense = 0;

//     const amountElements = month.querySelectorAll('.expense-amount');
//     amountElements.forEach(el => {
//       const amountText = el.textContent;
//       const amount = parseFloat(amountText.replace(/[.]/g, '').replace(',', '.').replace('€', ''));

//       if (el.classList.contains('amount-income')) {
//         totalIncome += amount;
//       } else if (el.classList.contains('amount-expense')) {
//         totalExpense += amount;
//       }
//     });

//     // Update the totals in the UI
//     const incomeEl = month.querySelector('.month-income span:last-child');
//     const expenseEl = month.querySelector('.month-expense span:last-child');

//     if (incomeEl) incomeEl.textContent = formatCurrency(totalIncome);
//     if (expenseEl) expenseEl.textContent = formatCurrency(totalExpense);
//   });
// }

// function formatCurrency(amount) {
//   return amount.toLocaleString('es-ES', {
//     minimumFractionDigits: 2,
//     maximumFractionDigits: 2
//   }) + '€';
// }

// function filterExpenses(searchTerm) {
//   const items = document.querySelectorAll('.expense-item');
//   let hasVisibleItems = false;

//   items.forEach(item => {
//     const text = item.textContent.toLowerCase();
//     if (text.includes(searchTerm)) {
//       item.style.display = '';
//       hasVisibleItems = true;
//     } else {
//       item.style.display = 'none';
//     }
//   });

//   // Show/hide empty state message
//   const emptyState = document.querySelector('.empty-state');
//   if (emptyState) {
//     emptyState.style.display = hasVisibleItems ? 'none' : 'block';
//   }
// }
