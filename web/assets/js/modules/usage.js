/**
 * Usage Module - Token usage tracking and display
 * Handles usage metrics, rate limits, and cost calculations
 */

(function() {
  'use strict';

  /**
   * Initialize usage panel
   */
  function initializeUsagePanel() {
    const usageToggle = document.getElementById('usage-toggle');
    const usagePanel = document.getElementById('usage-panel');
    
    if (usageToggle && usagePanel) {
      usageToggle.addEventListener('click', function() {
        const isCollapsed = usagePanel.classList.contains('collapsed');
        const toggleIcon = usageToggle.querySelector('.toggle-icon');
        
        if (isCollapsed) {
          usagePanel.classList.remove('collapsed');
          toggleIcon.textContent = '▼';
          // Load usage data when expanded
          const currentSessionId = window.AppState ? 
            window.AppState.getState('currentSessionId') : window.currentSessionId;
          
          if (currentSessionId) {
            loadSessionUsage(currentSessionId);
          }
          loadGlobalUsage();
        } else {
          usagePanel.classList.add('collapsed');
          toggleIcon.textContent = '▶';
        }
      });
    }
    
    // Load initial usage data if authenticated
    const currentSessionId = window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
    
    if (currentSessionId) {
      loadSessionUsage(currentSessionId);
    }
  }

  /**
   * Load usage data for current session
   * @param {string} sessionId - Session ID
   */
  async function loadSessionUsage(sessionId) {
    try {
      const response = await fetch(`/api/session/${sessionId}/usage`);
      if (response.ok) {
        const data = await response.json();
        updateSessionUsageDisplay(data);
      }
    } catch (error) {
      console.error('Failed to load session usage:', error);
    }
  }

  /**
   * Load global usage data
   */
  async function loadGlobalUsage() {
    try {
      const response = await fetch('/api/usage/global');
      if (response.ok) {
        const data = await response.json();
        updateGlobalUsageDisplay(data);
      }
    } catch (error) {
      console.error('Failed to load global usage:', error);
    }
    
    // Also load daily usage
    loadDailyUsage();
  }

  /**
   * Load daily usage data
   */
  async function loadDailyUsage() {
    try {
      const response = await fetch('/api/usage/daily');
      if (response.ok) {
        const data = await response.json();
        updateDailyUsageDisplay(data);
      }
    } catch (error) {
      console.error('Failed to load daily usage:', error);
    }
  }

  /**
   * Update session usage display
   * @param {Object} data - Usage data
   */
  function updateSessionUsageDisplay(data) {
    const inputTokensEl = document.getElementById('session-input-tokens');
    const outputTokensEl = document.getElementById('session-output-tokens');
    const costEl = document.getElementById('session-cost');
    
    if (inputTokensEl) inputTokensEl.textContent = formatTokenCount(data.usage.inputTokens);
    if (outputTokensEl) outputTokensEl.textContent = formatTokenCount(data.usage.outputTokens);
    if (costEl) costEl.textContent = `$${data.cost.total.toFixed(4)}`;
    
    // Update rate limits if available
    if (data.rateLimits) {
      updateRateLimitsDisplay(data.rateLimits);
    }
  }

  /**
   * Update global usage display
   * @param {Object} data - Global usage data
   */
  function updateGlobalUsageDisplay(data) {
    // Update quick info in header
    const quickInfo = document.getElementById('usage-quick-info');
    if (quickInfo) {
      const totalTokens = formatTokenCount(data.global.totalTokens);
      const totalCost = `$${data.global.totalCost.toFixed(2)}`;
      quickInfo.textContent = `${totalTokens} tokens | ${totalCost}`;
    }
    
    // Update warnings if any
    if (data.warnings && data.warnings.length > 0) {
      data.warnings.forEach(warning => {
        console.warn('Usage warning:', warning);
        // Could add visual warning indicator
      });
    }
  }

  /**
   * Update daily usage display
   * @param {Object} data - Daily usage data
   */
  function updateDailyUsageDisplay(data) {
    const dailyUsageEl = document.getElementById('daily-usage');
    if (dailyUsageEl) {
      const daily = data.daily;
      dailyUsageEl.innerHTML = `
        <div class="stat-item">
          <span class="stat-label">Today:</span>
          <span class="stat-value">${formatTokenCount(daily.totalTokens)} tokens</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">Cost:</span>
          <span class="stat-value">$${daily.totalCost.toFixed(4)}</span>
        </div>
      `;
    }
  }

  /**
   * Update rate limits display
   * @param {Object} rateLimits - Rate limit data
   */
  function updateRateLimitsDisplay(rateLimits) {
    if (!rateLimits) return;
    
    // Update request limits
    if (rateLimits.RequestsLimit > 0) {
      updateLimitBar('requests', rateLimits.RequestsRemaining, rateLimits.RequestsLimit);
    }
    
    // Update token limits
    if (rateLimits.InputTokensLimit > 0) {
      updateLimitBar('input-tokens', rateLimits.InputTokensRemaining, rateLimits.InputTokensLimit);
    }
    
    if (rateLimits.OutputTokensLimit > 0) {
      updateLimitBar('output-tokens', rateLimits.OutputTokensRemaining, rateLimits.OutputTokensLimit);
    }
  }

  /**
   * Update individual limit bar
   * @param {string} type - Type of limit
   * @param {number} remaining - Remaining amount
   * @param {number} limit - Total limit
   */
  function updateLimitBar(type, remaining, limit) {
    const progressEl = document.getElementById(`${type}-progress`);
    const textEl = document.getElementById(`${type}-remaining`);
    
    if (progressEl && textEl) {
      const percentage = (remaining / limit) * 100;
      progressEl.style.width = `${percentage}%`;
      
      // Change color based on percentage
      if (percentage < 20) {
        progressEl.className = 'progress-fill danger';
      } else if (percentage < 50) {
        progressEl.className = 'progress-fill warning';
      } else {
        progressEl.className = 'progress-fill';
      }
      
      // Format text based on type
      if (type === 'requests') {
        textEl.textContent = `${remaining} / ${limit}`;
      } else {
        textEl.textContent = `${formatTokenCount(remaining)} / ${formatTokenCount(limit)}`;
      }
    }
  }

  /**
   * Format token count for display
   * @param {number} count - Token count
   * @returns {string} Formatted count
   */
  function formatTokenCount(count) {
    if (count >= 1000000) {
      return `${(count / 1000000).toFixed(2)}M`;
    } else if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toString();
  }

  /**
   * Handle usage update events from SSE
   * @param {Object} data - Usage update data
   */
  function handleUsageUpdateEvent(data) {
    if (data.usage) {
      // Update session usage display
      const sessionData = {
        usage: data.usage,
        cost: calculateCostFromUsage(data.usage)
      };
      updateSessionUsageDisplay(sessionData);
    }
    
    if (data.rateLimits) {
      updateRateLimitsDisplay(data.rateLimits);
    }
    
    // Update current model if available
    if (data.model) {
      const modelEl = document.getElementById('current-model');
      if (modelEl) {
        modelEl.textContent = data.model;
      }
    }
  }

  /**
   * Calculate cost from usage (basic estimation)
   * @param {Object} usage - Usage data
   * @returns {Object} Cost breakdown
   */
  function calculateCostFromUsage(usage) {
    // Basic pricing estimates (adjust as needed)
    const inputRate = 0.000015; // $15 per million tokens (Opus)
    const outputRate = 0.000075; // $75 per million tokens (Opus)
    
    const inputCost = (usage.InputTokens || usage.inputTokens || 0) * inputRate;
    const outputCost = (usage.OutputTokens || usage.outputTokens || 0) * outputRate;
    
    return {
      input: inputCost,
      output: outputCost,
      total: inputCost + outputCost
    };
  }

  /**
   * Update usage display from SSE event
   * @param {Object} event - SSE event data
   */
  function updateUsageDisplay(usage) {
    if (!usage) return;
    
    // Update session usage
    const sessionData = {
      usage: {
        inputTokens: usage.InputTokens || usage.inputTokens || 0,
        outputTokens: usage.OutputTokens || usage.outputTokens || 0
      },
      cost: calculateCostFromUsage(usage)
    };
    
    updateSessionUsageDisplay(sessionData);
  }

  // Export to global scope
  window.UsageModule = {
    initializeUsagePanel,
    loadSessionUsage,
    loadGlobalUsage,
    loadDailyUsage,
    updateSessionUsageDisplay,
    updateGlobalUsageDisplay,
    updateDailyUsageDisplay,
    updateRateLimitsDisplay,
    updateLimitBar,
    formatTokenCount,
    handleUsageUpdateEvent,
    calculateCostFromUsage,
    updateUsageDisplay
  };

  // Also expose individual functions for backward compatibility
  window.initializeUsagePanel = initializeUsagePanel;
  window.loadSessionUsage = loadSessionUsage;
  window.loadGlobalUsage = loadGlobalUsage;
  window.loadDailyUsage = loadDailyUsage;
  window.updateSessionUsageDisplay = updateSessionUsageDisplay;
  window.updateGlobalUsageDisplay = updateGlobalUsageDisplay;
  window.updateDailyUsageDisplay = updateDailyUsageDisplay;
  window.updateRateLimitsDisplay = updateRateLimitsDisplay;
  window.updateLimitBar = updateLimitBar;
  window.formatTokenCount = formatTokenCount;
  window.handleUsageUpdateEvent = handleUsageUpdateEvent;
  window.calculateCostFromUsage = calculateCostFromUsage;
  window.updateUsageDisplay = updateUsageDisplay;

})();