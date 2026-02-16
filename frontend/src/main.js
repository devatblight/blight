import { IsFirstRun, CompleteOnboarding, Search, Execute, HideWindow, GetContextActions, ExecuteContextAction } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

class Blight {
    constructor() {
        this.selectedIndex = 0;
        this.results = [];
        this.debounceTimer = null;
        this.toastTimer = null;
        this.toastHovered = false;
        this.contextTarget = null;

        // Notification history
        this.notifications = [];

        this.searchInput = document.getElementById('search-input');
        this.resultsContainer = document.getElementById('results');
        this.splashEl = document.getElementById('splash');
        this.launcherEl = document.getElementById('app');
        this.contextMenuEl = document.getElementById('context-menu');

        // Notification elements
        this.notifIndicator = document.getElementById('notification-indicator');
        this.notifIcon = document.getElementById('notif-icon');
        this.notifText = document.getElementById('notif-text');
        this.notifHistory = document.getElementById('notification-history');
        this.notifHistoryList = document.getElementById('notif-history-list');
        this.notifClear = document.getElementById('notif-clear');

        this.init();
    }

    async init() {
        const firstRun = await IsFirstRun();
        if (firstRun) {
            this.showSplash();
        } else {
            this.showLauncher();
        }
    }

    showSplash() {
        this.splashEl.classList.remove('hidden');
        this.launcherEl.classList.add('hidden');
        this.initSplash();
    }

    showLauncher() {
        this.splashEl.classList.add('hidden');
        this.launcherEl.classList.remove('hidden');
        this.searchInput.focus();
        this.bindEvents();
        this.listenIndexStatus();
        this.bindNotificationUI();
        this.loadDefaultResults();
    }

    // --- Splash ---

    initSplash() {
        this.currentSlide = 0;

        document.getElementById('splash-next').addEventListener('click', () => {
            if (this.currentSlide < 3) this.goToSlide(this.currentSlide + 1);
        });

        document.getElementById('splash-skip').addEventListener('click', () => this.completeSplash());
        document.getElementById('splash-go').addEventListener('click', () => this.completeSplash());

        document.querySelectorAll('.splash-dot').forEach(dot => {
            dot.addEventListener('click', () => this.goToSlide(parseInt(dot.dataset.dot)));
        });
    }

    goToSlide(index) {
        document.querySelectorAll('.splash-slide').forEach((slide, i) => {
            slide.classList.remove('active', 'exit-left');
            if (i < index) slide.classList.add('exit-left');
            if (i === index) slide.classList.add('active');
        });

        document.querySelectorAll('.splash-dot').forEach((dot, i) => {
            dot.classList.toggle('active', i === index);
        });

        document.getElementById('splash-next').style.visibility = index >= 3 ? 'hidden' : 'visible';
        this.currentSlide = index;
    }

    async completeSplash() {
        await CompleteOnboarding('Alt+Space');
        this.splashEl.style.animation = 'splashOut 250ms ease forwards';
        setTimeout(() => this.showLauncher(), 250);
    }

    // --- Events ---

    bindEvents() {
        this.searchInput.addEventListener('input', () => this.onSearchInput());

        document.addEventListener('keydown', (e) => {
            if (!this.contextMenuEl.classList.contains('hidden')) {
                if (e.key === 'Escape') {
                    this.hideContextMenu();
                    e.preventDefault();
                }
                return;
            }

            switch (e.key) {
                case 'ArrowDown':
                    e.preventDefault();
                    this.moveSelection(1);
                    break;
                case 'ArrowUp':
                    e.preventDefault();
                    this.moveSelection(-1);
                    break;
                case 'Enter':
                    e.preventDefault();
                    this.executeSelected();
                    break;
                case 'Escape':
                    e.preventDefault();
                    if (this.searchInput.value) {
                        this.searchInput.value = '';
                        this.loadDefaultResults();
                    } else {
                        HideWindow();
                    }
                    break;
            }
        });

        document.addEventListener('click', (e) => {
            if (!this.contextMenuEl.contains(e.target)) {
                this.hideContextMenu();
            }
            // Close notification history if clicking outside
            if (this.notifHistory && !this.notifIndicator.contains(e.target) && !this.notifHistory.contains(e.target)) {
                this.notifHistory.classList.add('hidden');
            }
        });
    }

    onSearchInput() {
        clearTimeout(this.debounceTimer);
        this.debounceTimer = setTimeout(async () => {
            const query = this.searchInput.value.trim();
            this.results = await Search(query);
            this.selectedIndex = 0;
            this.renderResults();
        }, 120);
    }

    async loadDefaultResults() {
        this.results = await Search('');
        this.selectedIndex = 0;
        this.renderResults();
    }

    moveSelection(delta) {
        if (this.results.length === 0) return;
        this.selectedIndex = (this.selectedIndex + delta + this.results.length) % this.results.length;
        this.renderResults();
        const selected = this.resultsContainer.querySelector('.result-item.selected');
        if (selected) selected.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }

    async executeSelected() {
        if (this.results.length === 0) return;
        const result = this.results[this.selectedIndex];

        if (result.id === 'calc-result') {
            await navigator.clipboard.writeText(result.title);
            this.showToast('Copied result', result.title);
            return;
        }

        const response = await Execute(result.id);
        if (response === 'copied') {
            this.showToast('Copied to clipboard', result.title);
        } else if (response === 'ok') {
            if (result.id.startsWith('sys-')) {
                this.showToast(result.title, result.subtitle);
            } else {
                this.showToast(`Launched ${result.title}`, result.path || '');
            }
        }
    }

    // --- Rendering ---

    renderResults() {
        if (this.results.length === 0) {
            this.resultsContainer.innerHTML = `
                <div class="no-results">
                    <div style="font-size: 24px; opacity: 0.3;">⌕</div>
                    <div>No results found</div>
                </div>
            `;
            return;
        }

        let html = '';
        let lastCategory = '';

        this.results.forEach((result, index) => {
            if (result.category !== lastCategory) {
                html += `<div class="result-category">${result.category}</div>`;
                lastCategory = result.category;
            }

            const selected = index === this.selectedIndex ? 'selected' : '';
            const iconHtml = result.icon && result.icon.startsWith('data:')
                ? `<div class="result-icon"><img src="${result.icon}" alt=""/></div>`
                : `<div class="result-icon-fallback">●</div>`;

            html += `
                <div class="result-item ${selected}" data-index="${index}" data-id="${result.id}">
                    ${iconHtml}
                    <div class="result-text">
                        <div class="result-title">${result.title}</div>
                        <div class="result-subtitle">${result.subtitle}</div>
                    </div>
                    <div class="result-badge">${result.category}</div>
                </div>
            `;
        });

        this.resultsContainer.innerHTML = html;

        this.resultsContainer.querySelectorAll('.result-item').forEach(item => {
            item.addEventListener('click', () => {
                this.selectedIndex = parseInt(item.dataset.index);
                this.renderResults();
                this.executeSelected();
            });

            item.addEventListener('mouseenter', () => {
                this.selectedIndex = parseInt(item.dataset.index);
                this.renderResults();
            });

            item.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                this.selectedIndex = parseInt(item.dataset.index);
                this.renderResults();
                this.showContextMenu(e.clientX, e.clientY, item.dataset.id);
            });
        });
    }

    // --- Context Menu ---

    async showContextMenu(x, y, resultId) {
        this.contextTarget = resultId;
        const actions = await GetContextActions(resultId);

        let html = '';
        actions.forEach(action => {
            html += `
                <button class="context-action" data-action="${action.id}">
                    <span class="context-action-icon">${action.icon}</span>
                    ${action.label}
                </button>
            `;
        });

        this.contextMenuEl.innerHTML = html;
        this.contextMenuEl.classList.remove('hidden');

        const rect = this.contextMenuEl.getBoundingClientRect();
        const maxX = window.innerWidth - rect.width - 8;
        const maxY = window.innerHeight - rect.height - 8;
        this.contextMenuEl.style.left = `${Math.min(x, maxX)}px`;
        this.contextMenuEl.style.top = `${Math.min(y, maxY)}px`;

        this.contextMenuEl.querySelectorAll('.context-action').forEach(btn => {
            btn.addEventListener('click', async () => {
                const actionId = btn.dataset.action;
                const response = await ExecuteContextAction(this.contextTarget, actionId);
                this.hideContextMenu();

                if (actionId === 'copy-path') {
                    this.showToast('Path copied', 'Copied to clipboard');
                } else if (response === 'ok' && actionId !== 'explorer') {
                    this.showToast(`Launched`, this.contextTarget);
                }
            });
        });
    }

    hideContextMenu() {
        this.contextMenuEl.classList.add('hidden');
        this.contextTarget = null;
    }

    // --- Toast (left side of footer) ---

    showToast(message, detail = '') {
        const brand = document.getElementById('footer-brand');
        const toastEl = document.getElementById('footer-toast');

        brand.classList.add('hidden-by-toast');

        toastEl.textContent = message;
        toastEl.classList.add('visible');

        this.toastHovered = false;

        toastEl.onmouseenter = () => {
            this.toastHovered = true;
            clearTimeout(this.toastTimer);
        };

        toastEl.onmouseleave = () => {
            this.toastHovered = false;
            this.startToastDismiss(brand, toastEl);
        };

        clearTimeout(this.toastTimer);
        this.startToastDismiss(brand, toastEl);
    }

    startToastDismiss(brand, toastEl) {
        this.toastTimer = setTimeout(() => {
            if (this.toastHovered) return;
            toastEl.classList.remove('visible');
            brand.classList.remove('hidden-by-toast');
        }, 5000);
    }

    // --- Notification Indicator (bottom-right) ---

    listenIndexStatus() {
        EventsOn('indexStatus', (status) => {
            const stateIcons = {
                checking: '🔍',
                indexing: '📁',
                ready: '✓',
                idle: '—',
            };
            const icon = stateIcons[status.state] || '';
            this.setNotification(icon, status.message, status.state);
        });
    }

    setNotification(icon, message, state) {
        this.notifIcon.textContent = icon;
        this.notifText.textContent = message;

        // Add to history
        this.notifications.unshift({
            icon,
            message,
            state,
            time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
        });

        // Keep max 20 notifications
        if (this.notifications.length > 20) {
            this.notifications = this.notifications.slice(0, 20);
        }

        this.renderNotificationHistory();
    }

    bindNotificationUI() {
        // Toggle history on click
        this.notifIndicator.addEventListener('click', (e) => {
            e.stopPropagation();
            this.notifHistory.classList.toggle('hidden');
        });

        // Clear button
        if (this.notifClear) {
            this.notifClear.addEventListener('click', () => {
                this.notifications = [];
                this.renderNotificationHistory();
            });
        }

        // Show history on hover
        this.notifIndicator.addEventListener('mouseenter', () => {
            if (this.notifications.length > 0) {
                this.notifHistory.classList.remove('hidden');
            }
        });

        // Hide when mouse leaves the whole area
        const footer = this.notifIndicator.closest('.footer');
        footer.addEventListener('mouseleave', () => {
            this.notifHistory.classList.add('hidden');
        });
    }

    renderNotificationHistory() {
        if (!this.notifHistoryList) return;

        if (this.notifications.length === 0) {
            this.notifHistoryList.innerHTML = '<div class="notif-history-empty">No notifications</div>';
            return;
        }

        this.notifHistoryList.innerHTML = this.notifications.map(n => `
            <div class="notif-history-item">
                <span class="notif-h-icon">${n.icon}</span>
                <div class="notif-h-text">
                    <div class="notif-h-msg">${n.message}</div>
                    <div class="notif-h-time">${n.time}</div>
                </div>
            </div>
        `).join('');
    }
}

document.addEventListener('DOMContentLoaded', () => new Blight());
