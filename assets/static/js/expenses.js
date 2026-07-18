/**
 * Main entry point for ExpenseTrace frontend functionality
 *
 * This file imports and initializes all application modules:
 * - Toggle system for collapsible UI elements
 * - Tab switching for category patterns
 * - Financial bar chart with savings visualization
 * - Donut charts for category breakdown
 */

// Import all modules
import { initializeToggleSystem } from './toggle.js';
import { initializeTabSystem } from './tabs.js';
import { initializeBarChart } from './charts/bar-chart.js';
import { initializeDonutChart } from './charts/donut-chart.js';


// Initialize all systems when DOM is ready
document.addEventListener('DOMContentLoaded', function () {
  // Core UI systems
  initializeToggleSystem();
  initializeTabSystem();

  // Charts (these check for their containers before initializing)
  initializeBarChart();
  initializeDonutChart();
});
