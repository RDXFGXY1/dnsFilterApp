// Dashboard JavaScript

// Auto-refresh interval (5 seconds)
const REFRESH_INTERVAL = 5000;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    loadStats();
    loadTopBlocked();
    loadRecentQueries();
    loadWhitelist();
    
    // Auto-refresh
    setInterval(() => {
        loadStats();
        loadRecentQueries();
    }, REFRESH_INTERVAL);
});

// Load statistics
async function loadStats() {
    try {
        const response = await fetch('/api/stats/blocked');
        const stats = await response.json();
        
        document.getElementById('total-queries').textContent = 
            stats.total_blocked || '0';
        document.getElementById('blocked-queries').textContent = 
            stats.total_blocked || '0';
        document.getElementById('block-rate').textContent = 
            calculateBlockRate(stats) + '%';
        
        const countResponse = await fetch('/api/blocklist/count');
        const countData = await countResponse.json();
        document.getElementById('blocked-domains').textContent = 
            countData.count?.toLocaleString() || '0';
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// Load top blocked domains
async function loadTopBlocked() {
    try {
        const response = await fetch('/api/stats/top-blocked');
        const topBlocked = await response.json();
        
        const container = document.getElementById('top-blocked-list');
        container.innerHTML = '';
        
        if (Object.keys(topBlocked).length === 0) {
            container.innerHTML = '<p class="no-data">No blocked queries yet</p>';
            return;
        }
        
        const list = document.createElement('ul');
        list.className = 'top-blocked-list';
        
        Object.entries(topBlocked)
            .sort((a, b) => b[1] - a[1])
            .forEach(([domain, count]) => {
                const li = document.createElement('li');
                li.innerHTML = `
                    <span class="domain">${escapeHtml(domain)}</span>
                    <span class="count">${count} attempts</span>
                `;
                list.appendChild(li);
            });
        
        container.appendChild(list);
    } catch (error) {
        console.error('Failed to load top blocked:', error);
    }
}

// Load recent queries
async function loadRecentQueries() {
    try {
        const response = await fetch('/api/recent');
        const queries = await response.json();
        
        const tbody = document.querySelector('#recent-queries tbody');
        tbody.innerHTML = '';
        
        if (queries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="3" class="no-data">No recent blocked queries</td></tr>';
            return;
        }
        
        queries.forEach(query => {
            const row = tbody.insertRow();
            row.innerHTML = `
                <td>${formatTime(query.timestamp)}</td>
                <td>${escapeHtml(query.domain)}</td>
                <td>${escapeHtml(query.client_ip)}</td>
            `;
        });
    } catch (error) {
        console.error('Failed to load recent queries:', error);
    }
}

// Load whitelist
async function loadWhitelist() {
    try {
        const response = await fetch('/api/whitelist');
        const whitelist = await response.json();
        
        const container = document.getElementById('whitelist-items');
        container.innerHTML = '';
        
        if (whitelist.length === 0) {
            container.innerHTML = '<p class="no-data">No whitelisted domains</p>';
            return;
        }
        
        const list = document.createElement('ul');
        list.className = 'whitelist';
        
        whitelist.forEach(domain => {
            const li = document.createElement('li');
            li.innerHTML = `
                <span>${escapeHtml(domain)}</span>
                <button onclick="removeFromWhitelist('${escapeHtml(domain)}')" class="btn-remove">Remove</button>
            `;
            list.appendChild(li);
        });
        
        container.appendChild(list);
    } catch (error) {
        console.error('Failed to load whitelist:', error);
    }
}

// Add to whitelist
async function addToWhitelist() {
    const input = document.getElementById('whitelist-domain');
    const domain = input.value.trim();
    
    if (!domain) {
        alert('Please enter a domain');
        return;
    }
    
    try {
        const response = await fetch('/api/whitelist', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ domain }),
        });
        
        if (response.ok) {
            input.value = '';
            loadWhitelist();
            showNotification('Domain added to whitelist');
        } else {
            alert('Failed to add domain to whitelist');
        }
    } catch (error) {
        console.error('Failed to add to whitelist:', error);
        alert('Failed to add domain to whitelist');
    }
}

// Remove from whitelist
async function removeFromWhitelist(domain) {
    if (!confirm(`Remove ${domain} from whitelist?`)) {
        return;
    }
    
    try {
        const response = await fetch(`/api/whitelist/${encodeURIComponent(domain)}`, {
            method: 'DELETE',
        });
        
        if (response.ok) {
            loadWhitelist();
            showNotification('Domain removed from whitelist');
        } else {
            alert('Failed to remove domain from whitelist');
        }
    } catch (error) {
        console.error('Failed to remove from whitelist:', error);
        alert('Failed to remove domain from whitelist');
    }
}

// Update blocklists
async function updateBlocklists() {
    if (!confirm('This will download and update all blocklists. Continue?')) {
        return;
    }
    
    try {
        const response = await fetch('/api/blocklist/update', {
            method: 'POST',
        });
        
        if (response.ok) {
            showNotification('Blocklist update started. This may take a few minutes.');
            setTimeout(loadStats, 3000);
        } else {
            alert('Failed to start blocklist update');
        }
    } catch (error) {
        console.error('Failed to update blocklists:', error);
        alert('Failed to start blocklist update');
    }
}

// Clear cache
async function clearCache() {
    if (!confirm('Clear DNS cache?')) {
        return;
    }
    
    try {
        const response = await fetch('/api/system/clear-cache', {
            method: 'POST',
        });
        
        if (response.ok) {
            showNotification('DNS cache cleared');
        } else {
            alert('Failed to clear cache');
        }
    } catch (error) {
        console.error('Failed to clear cache:', error);
        alert('Failed to clear cache');
    }
}

// Export logs
function exportLogs() {
    showNotification('Log export feature coming soon');
}

// Utility functions
function calculateBlockRate(stats) {
    if (!stats.total_blocked) return 0;
    return ((stats.total_blocked / (stats.total_blocked + 1000)) * 100).toFixed(2);
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showNotification(message) {
    // Simple notification (can be enhanced with a toast library)
    const notification = document.createElement('div');
    notification.className = 'notification';
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('show');
    }, 100);
    
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}
