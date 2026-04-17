import {
    GetConfig,
    SaveSettings,
    GetVersion,
    GetStartupEnabled,
    GetDataDir,
    GetInstallDir,
    OpenFolder,
    OpenFolderPicker,
    ReindexFiles,
    ClearIndex,
    CancelIndex,
    CheckForUpdates,
    Uninstall,
    CloseSettings,
    ExportSettings,
    ImportSettings,
    GetAliases,
    SaveAlias,
    DeleteAlias,
    OpenURL,
} from '../../wailsjs/go/main/App';
import { marked, Renderer } from 'marked';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { main, files } from '../../wailsjs/go/models';
import { escapeHtml, inputEl, selectEl } from './utils';
import { ToastType } from './toast';
import { showConfirmModal } from './modal';

export interface SettingsDeps {
    showToast: (msg: string, detail?: string, type?: ToastType) => void;
    applyRuntimeSettings: (cfg: main.BlightConfig) => void;
    onClose: () => void;
    settingsMode: boolean;
    getLastUpdateCheck: () => number;
    setLastUpdateCheck: (t: number) => void;
    onUpdateAvailable: (update: main.UpdateInfo) => void;
}

export class Settings {
    private panelEl: HTMLElement;
    private deps: SettingsDeps;
    private currentIndexDirs: string[] = [];
    private lastUpdateCheck = 0;

    // Hotkey recorder state
    private _hkPending = '';
    private _hkKeydownFn: ((e: KeyboardEvent) => void) | null = null;
    private _hkKeyupFn: ((e: KeyboardEvent) => void) | null = null;

    constructor(panelEl: HTMLElement, deps: SettingsDeps) {
        this.panelEl = panelEl;
        this.deps = deps;
    }

    setSettingsMode(value: boolean): void {
        this.deps.settingsMode = value;
    }

    get isOpen(): boolean {
        return !this.panelEl.classList.contains('hidden');
    }

    async open(): Promise<void> {
        this.panelEl.classList.remove('hidden');
        this.panelEl.style.animation = 'none';
        void this.panelEl.offsetHeight; // force reflow
        this.panelEl.style.animation = '';

        this.activateTab('general');

        try {
            const [config, version, startupEnabled] = await Promise.all([
                GetConfig(),
                GetVersion(),
                GetStartupEnabled(),
            ]);

            // General tab
            const hotkeyDisplay = document.getElementById('settings-hotkey-display');
            if (hotkeyDisplay) hotkeyDisplay.textContent = config.hotkey || 'Alt+Space';

            const lastQueryMode = selectEl('settings-last-query-mode');
            if (lastQueryMode) lastQueryMode.value = config.lastQueryMode || 'clear';

            const hideDeactivated = inputEl('settings-hide-deactivated');
            if (hideDeactivated) hideDeactivated.checked = config.hideWhenDeactivated !== false;

            const windowPosition = selectEl('settings-window-position');
            if (windowPosition) windowPosition.value = config.windowPosition || 'center';

            const clipSizeInput = inputEl('settings-clipboard-size');
            if (clipSizeInput) clipSizeInput.value = String(config.maxClipboard || 50);

            // Search tab
            const maxResults = inputEl('settings-max-results');
            if (maxResults) maxResults.value = String(config.maxResults || 8);

            const searchDelay = inputEl('settings-search-delay');
            if (searchDelay) searchDelay.value = String(config.searchDelay || 120);

            const placeholderText = inputEl('settings-placeholder-text');
            if (placeholderText) placeholderText.value = config.placeholderText || '';

            const searchEngineURL = inputEl('settings-search-engine-url');
            if (searchEngineURL)
                searchEngineURL.value =
                    config.searchEngineURL || 'https://www.google.com/search?q=%s';

            const showPlaceholder = inputEl('settings-show-placeholder');
            if (showPlaceholder) showPlaceholder.checked = config.showPlaceholder !== false;

            // Appearance tab
            const theme = selectEl('settings-theme');
            if (theme) theme.value = config.theme || 'dark';

            const useAnimation = inputEl('settings-use-animation');
            if (useAnimation) useAnimation.checked = config.useAnimation !== false;

            const footerHints = selectEl('settings-footer-hints');
            if (footerHints) footerHints.value = config.footerHints || 'always';

            // System tab
            const startOnStartup = inputEl('settings-start-on-startup');
            if (startOnStartup) startOnStartup.checked = startupEnabled;

            const hideNotifyIcon = inputEl('settings-hide-notify-icon');
            if (hideNotifyIcon) hideNotifyIcon.checked = !!config.hideNotifyIcon;

            // Files tab
            const includeFolders = inputEl('settings-include-folders');
            if (includeFolders) includeFolders.checked = !config.disableFolderIndex;

            // Updates tab
            const versionEl = document.getElementById('settings-version');
            if (versionEl) versionEl.textContent = `v${version}`;

            // Misc tab
            GetDataDir()
                .then((d) => {
                    const el = document.getElementById('misc-data-dir');
                    if (el) el.textContent = d;
                })
                .catch(() => {});
            GetInstallDir()
                .then((d) => {
                    const el = document.getElementById('misc-install-dir');
                    if (el) el.textContent = d;
                })
                .catch(() => {});

            this.currentIndexDirs = config.indexDirs || [];
            this._renderIndexDirs();

            // Aliases tab
            this._loadAliasesTab();
        } catch (e) {
            // eslint-disable-next-line no-console
            console.error('Failed to load settings:', e);
        }
    }

    close(): void {
        if (this.deps.settingsMode) {
            CloseSettings();
            return;
        }
        this.panelEl.classList.add('hidden');
        this.deps.onClose();
    }

    activateTab(name: string): void {
        document.querySelectorAll<HTMLElement>('.settings-nav-item').forEach((btn) => {
            const isActive = btn.dataset['tab'] === name;
            btn.classList.toggle('active', isActive);
            btn.setAttribute('aria-selected', String(isActive));
        });
        document.querySelectorAll('.settings-tab').forEach((tab) => {
            tab.classList.toggle('hidden', tab.id !== `tab-${name}`);
        });
    }

    bind(): void {
        document.querySelectorAll<HTMLElement>('.settings-nav-item').forEach((btn) => {
            btn.addEventListener('click', () => this.activateTab(btn.dataset['tab'] ?? ''));
        });
        this._bindTabKeyNav();
        this._bindAliasAdd();
        this._bindHotkeyBadge();

        document.getElementById('settings-close')?.addEventListener('click', () => this.close());
        document.getElementById('settings-cancel')?.addEventListener('click', () => this.close());

        const saveBtn = document.getElementById('settings-save');
        if (saveBtn) {
            saveBtn.addEventListener('click', async () => {
                const cfg = {
                    firstRun: false,
                    hotkey:
                        document.getElementById('settings-hotkey-display')?.textContent ||
                        'Alt+Space',
                    maxClipboard: parseInt(inputEl('settings-clipboard-size')?.value || '50', 10),
                    lastQueryMode: selectEl('settings-last-query-mode')?.value || 'clear',
                    hideWhenDeactivated: inputEl('settings-hide-deactivated')?.checked ?? true,
                    windowPosition: selectEl('settings-window-position')?.value || 'center',
                    maxResults: parseInt(inputEl('settings-max-results')?.value || '8', 10),
                    searchDelay: parseInt(inputEl('settings-search-delay')?.value || '120', 10),
                    placeholderText: inputEl('settings-placeholder-text')?.value || '',
                    showPlaceholder: inputEl('settings-show-placeholder')?.checked ?? true,
                    theme: selectEl('settings-theme')?.value || 'dark',
                    useAnimation: inputEl('settings-use-animation')?.checked ?? true,
                    footerHints: selectEl('settings-footer-hints')?.value || 'always',
                    startOnStartup: inputEl('settings-start-on-startup')?.checked ?? false,
                    hideNotifyIcon: inputEl('settings-hide-notify-icon')?.checked ?? false,
                    disableFolderIndex: !(inputEl('settings-include-folders')?.checked ?? true),
                    searchEngineURL: inputEl('settings-search-engine-url')?.value?.trim() || '',
                    indexDirs: this.currentIndexDirs,
                };
                try {
                    const cfgObj = main.BlightConfig.createFrom(cfg);
                    await SaveSettings(cfgObj);
                    this.deps.applyRuntimeSettings(cfgObj);
                    if (this.deps.settingsMode) {
                        CloseSettings();
                        return;
                    }
                    this.deps.showToast('Settings saved', 'Changes applied', 'success');
                    this.close();
                } catch (e) {
                    this.deps.showToast('Save failed', String(e), 'error');
                }
            });
        }

        // Files / indexing
        document.getElementById('settings-reindex')?.addEventListener('click', async () => {
            await ReindexFiles();
            const statusEl = document.getElementById('settings-index-status');
            if (statusEl) statusEl.textContent = 'Reindexing…';
        });

        document
            .getElementById('settings-cancel-index')
            ?.addEventListener('click', () => CancelIndex());

        document.getElementById('settings-clear-index')?.addEventListener('click', async () => {
            await ClearIndex();
            const statusEl = document.getElementById('settings-index-status');
            if (statusEl) statusEl.textContent = 'Index cleared';
            this.deps.showToast('Index cleared', '');
        });

        document.getElementById('settings-add-dir')?.addEventListener('click', async () => {
            const dir = await OpenFolderPicker();
            if (dir) {
                this.currentIndexDirs = [...this.currentIndexDirs, dir];
                this._renderIndexDirs();
            }
        });

        EventsOn('indexStatus', (status: files.IndexStatus) => {
            const statusEl = document.getElementById('settings-index-status');
            if (statusEl) statusEl.textContent = status.message;
            const reindexBtn = document.getElementById(
                'settings-reindex'
            ) as HTMLButtonElement | null;
            const cancelBtn = document.getElementById('settings-cancel-index');
            const indexing = status.state === 'indexing';
            if (reindexBtn) reindexBtn.disabled = indexing;
            if (cancelBtn) cancelBtn.classList.toggle('hidden', !indexing);
        });

        // Updates tab
        const checkUpdatesBtn = document.getElementById(
            'settings-check-updates'
        ) as HTMLButtonElement | null;
        const updateStatus = document.getElementById('settings-update-status');
        if (checkUpdatesBtn) {
            checkUpdatesBtn.addEventListener('click', async () => {
                const cooldown = 10000;
                const elapsed = Date.now() - this.deps.getLastUpdateCheck();
                if (elapsed < cooldown) {
                    const remaining = Math.ceil((cooldown - elapsed) / 1000);
                    if (updateStatus) {
                        updateStatus.textContent = `Please wait ${remaining}s before checking again`;
                        updateStatus.className = 'settings-update-status error';
                    }
                    return;
                }
                this.deps.setLastUpdateCheck(Date.now());
                checkUpdatesBtn.disabled = true;
                checkUpdatesBtn.textContent = 'Checking…';
                if (updateStatus) {
                    updateStatus.textContent = '';
                    updateStatus.className = 'settings-update-status';
                }
                try {
                    const update = await CheckForUpdates();
                    if (update && update.available) {
                        if (updateStatus) {
                            updateStatus.textContent = '';
                            updateStatus.className = 'settings-update-status';
                        }
                        this.deps.onUpdateAvailable(update);
                    } else if (update && update.error) {
                        if (updateStatus) {
                            updateStatus.textContent = update.error;
                            updateStatus.className = 'settings-update-status error';
                        }
                    } else {
                        if (updateStatus) {
                            updateStatus.innerHTML =
                                '<span class="update-status-ok">&#x2713; You\'re on the latest version</span>';
                            updateStatus.className = 'settings-update-status success';
                        }
                    }
                } catch (e) {
                    if (updateStatus) {
                        updateStatus.textContent = String(e);
                        updateStatus.className = 'settings-update-status error';
                    }
                } finally {
                    checkUpdatesBtn.disabled = false;
                    checkUpdatesBtn.textContent = 'Check for Updates';
                }
            });
        }

        // Updates tab – open external links in the default browser via the backend
        document
            .querySelectorAll<HTMLAnchorElement>('.update-res-link, .update-res-github')
            .forEach((a) => {
                a.addEventListener('click', (e) => {
                    e.preventDefault();
                    const url = a.href;
                    if (url) OpenURL(url);
                });
            });
        // Release notes: event delegation for dynamically-rendered rn-link anchors
        document.getElementById('settings-update-notes')?.addEventListener('click', (e) => {
            const target = (e.target as HTMLElement).closest<HTMLAnchorElement>('a.rn-link');
            if (target) {
                e.preventDefault();
                const url = target.href;
                if (url) OpenURL(url);
            }
        });

        // Misc tab
        document.getElementById('misc-open-data')?.addEventListener('click', async () => {
            const dir = await GetInstallDir();
            OpenFolder(dir);
        });
        document.getElementById('misc-open-install')?.addEventListener('click', async () => {
            const dir = await GetInstallDir();
            OpenFolder(dir);
        });
        document.getElementById('misc-uninstall')?.addEventListener('click', () => {
            showConfirmModal(
                'Uninstall blight?',
                'This will permanently remove blight from your system. Your config and data in .blight will not be deleted.',
                'Uninstall',
                true,
                async () => {
                    const res = await Uninstall();
                    if (res !== 'success') {
                        this.deps.showToast(
                            'Uninstall failed',
                            res
                                .replace('not-found:', 'Uninstaller not found: ')
                                .replace('error:', ''),
                            'error'
                        );
                    }
                }
            );
        });

        // Export / Import
        const exportBtn = document.getElementById('misc-export-settings');
        if (exportBtn) {
            exportBtn.addEventListener('click', async () => {
                try {
                    const json = await ExportSettings();
                    const blob = new Blob([json], { type: 'application/json' });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = 'blight-settings.json';
                    a.click();
                    URL.revokeObjectURL(url);
                    this.deps.showToast('Settings exported', 'blight-settings.json', 'success');
                } catch (e) {
                    this.deps.showToast('Export failed', String(e), 'error');
                }
            });
        }

        const importFileInput = document.getElementById(
            'misc-import-file'
        ) as HTMLInputElement | null;
        const importBtn = document.getElementById('misc-import-settings');
        if (importBtn && importFileInput) {
            importBtn.addEventListener('click', () => importFileInput.click());
            importFileInput.addEventListener('change', async () => {
                const file = importFileInput.files?.[0];
                if (!file) return;
                try {
                    const text = await file.text();
                    showConfirmModal(
                        'Import settings?',
                        'This will overwrite your current configuration. blight will reload the new settings immediately.',
                        'Import',
                        false,
                        async () => {
                            await ImportSettings(text);
                            this.deps.showToast(
                                'Settings imported',
                                'Reload blight to apply fully',
                                'success'
                            );
                        }
                    );
                } catch (e) {
                    this.deps.showToast('Import failed', String(e), 'error');
                }
                importFileInput.value = '';
            });
        }
    }

    updateIndexStatus(msg: string): void {
        const el = document.getElementById('settings-index-status');
        if (el) el.textContent = msg;
    }

    showUpdateInstallRow(update: main.UpdateInfo, onInstall: () => void): void {
        const row = document.getElementById('settings-update-install-row');
        const label = document.getElementById('settings-update-version-label');
        const notesEl = document.getElementById('settings-update-notes');
        const githubLink = document.getElementById(
            'settings-update-github-link'
        ) as HTMLAnchorElement | null;
        const installBtn = document.getElementById(
            'settings-install-update'
        ) as HTMLButtonElement | null;

        if (row) row.classList.remove('hidden');
        if (label) label.textContent = `v${update.version}`;
        if (notesEl) notesEl.innerHTML = this._renderReleaseNotes(update.notes);
        if (githubLink) {
            const releaseUrl = `https://github.com/devatblight/blight/releases/tag/v${update.version}`;
            githubLink.href = releaseUrl;
            githubLink.onclick = (e) => {
                e.preventDefault();
                OpenURL(releaseUrl);
            };
        }
        if (installBtn) installBtn.onclick = onInstall;
    }

    // Converts GitHub release Markdown to HTML using the `marked` library (GFM).
    // Links are forced to open in a new tab (Wails intercepts target="_blank").
    private _renderReleaseNotes(raw: string): string {
        if (!raw || !raw.trim()) return '<span class="rn-empty">No release notes available.</span>';

        const renderer = new Renderer();

        // Open all links in the system browser via Wails target="_blank" interception
        renderer.link = ({ href, title, text }) => {
            const safeHref = (href || '').replace(/"/g, '&quot;');
            const titleAttr = title ? ` title="${title.replace(/"/g, '&quot;')}"` : '';
            return `<a href="${safeHref}"${titleAttr} target="_blank" class="rn-link">${text}</a>`;
        };

        return marked.parse(raw, {
            renderer,
            gfm: true,
            breaks: true,
        }) as string;
    }

    private _renderIndexDirs(): void {
        const container = document.getElementById('settings-index-dirs');
        if (!container) return;
        const dirs = this.currentIndexDirs;
        if (dirs.length === 0) {
            container.innerHTML =
                '<div style="font-size:11px;color:var(--text-tertiary)">No extra directories added</div>';
            return;
        }
        container.innerHTML = dirs
            .map(
                (d, i) => `
            <div class="settings-dir-item">
                <span class="settings-dir-path">${escapeHtml(d)}</span>
                <button class="settings-dir-remove" data-index="${i}">✕</button>
            </div>
        `
            )
            .join('');
        container.querySelectorAll<HTMLElement>('.settings-dir-remove').forEach((btn) => {
            btn.addEventListener('click', () => {
                const idx = parseInt(btn.dataset['index'] ?? '0', 10);
                this.currentIndexDirs = this.currentIndexDirs.filter((_, i) => i !== idx);
                this._renderIndexDirs();
            });
        });
    }

    private async _loadAliasesTab(): Promise<void> {
        try {
            const aliases = await GetAliases();
            this._renderAliases(aliases);
        } catch {
            /* non-critical */
        }
    }

    private _renderAliases(aliases: Record<string, string>): void {
        const list = document.getElementById('aliases-list');
        if (!list) return;
        const entries = Object.entries(aliases);
        if (entries.length === 0) {
            list.innerHTML =
                '<div style="font-size:11px;color:var(--text-tertiary);padding:8px 0">No aliases yet. Add one above.</div>';
            return;
        }
        list.innerHTML = entries
            .map(
                ([trigger, expansion]) => `
            <div class="alias-item">
                <span class="alias-trigger">${escapeHtml(trigger)}</span>
                <span class="alias-arrow">→</span>
                <span class="alias-expansion" title="${escapeHtml(expansion)}">${escapeHtml(expansion)}</span>
                <button class="alias-remove" data-trigger="${escapeHtml(trigger)}" title="Delete alias">✕</button>
            </div>
        `
            )
            .join('');
        list.querySelectorAll<HTMLElement>('.alias-remove').forEach((btn) => {
            btn.addEventListener('click', async () => {
                const trigger = btn.dataset['trigger'] ?? '';
                await DeleteAlias(trigger);
                await this._loadAliasesTab();
                this.deps.showToast('Alias deleted', trigger, 'info');
            });
        });
    }

    private _bindAliasAdd(): void {
        const addBtn = document.getElementById('alias-add-btn');
        const triggerInput = document.getElementById(
            'alias-trigger-input'
        ) as HTMLInputElement | null;
        const expansionInput = document.getElementById(
            'alias-expansion-input'
        ) as HTMLInputElement | null;
        if (!addBtn || !triggerInput || !expansionInput) return;

        const doAdd = async () => {
            const trigger = triggerInput.value.trim();
            const expansion = expansionInput.value.trim();
            if (!trigger || !expansion) {
                this.deps.showToast('Both fields required', '', 'warning');
                return;
            }
            try {
                await SaveAlias(trigger, expansion);
                triggerInput.value = '';
                expansionInput.value = '';
                await this._loadAliasesTab();
                this.deps.showToast(`Alias "${trigger}" saved`, expansion, 'success');
            } catch (e) {
                this.deps.showToast('Save failed', String(e), 'error');
            }
        };

        addBtn.addEventListener('click', doAdd);
        expansionInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') doAdd();
        });
    }

    private _bindHotkeyBadge(): void {
        document.getElementById('settings-hotkey-edit')?.addEventListener('click', () => {
            const current =
                document.getElementById('settings-hotkey-display')?.textContent || 'Alt+Space';
            this._openHotkeyRecorder(current);
        });
    }

    private _openHotkeyRecorder(currentHotkey: string): void {
        const modal = document.getElementById('hotkey-modal')!;
        const canvas = document.getElementById('hotkey-canvas')!;
        const saveBtn = document.getElementById('hotkey-save-btn') as HTMLElement & {
            disabled: boolean;
        };
        const clearBtn = document.getElementById('hotkey-clear-btn')!;
        const cancelBtn = document.getElementById('hotkey-cancel-btn')!;
        const currentValEl = document.getElementById('hotkey-modal-current-val')!;

        currentValEl.textContent = currentHotkey;
        this._hkPending = '';
        saveBtn.disabled = true;
        this._renderHkCanvas(null, false);

        modal.classList.remove('hidden');
        canvas.classList.add('hk-active');
        canvas.focus();

        const close = () => {
            modal.classList.add('hidden');
            canvas.classList.remove('hk-active');
            if (this._hkKeydownFn) document.removeEventListener('keydown', this._hkKeydownFn, true);
            if (this._hkKeyupFn) document.removeEventListener('keyup', this._hkKeyupFn, true);
            this._hkKeydownFn = null;
            this._hkKeyupFn = null;
        };

        // Close on overlay click
        modal.onclick = (e: MouseEvent) => {
            if (e.target === modal) close();
        };
        cancelBtn.onclick = () => close();

        clearBtn.onclick = () => {
            this._hkPending = '';
            saveBtn.disabled = true;
            this._renderHkCanvas(null, false);
            canvas.focus();
        };

        saveBtn.onclick = () => {
            if (this._hkPending) {
                const display = document.getElementById('settings-hotkey-display')!;
                display.textContent = this._hkPending;
                close();
            }
        };

        this._hkKeydownFn = (e: KeyboardEvent) => {
            e.preventDefault();
            e.stopImmediatePropagation();

            const mods: string[] = [];
            if (e.ctrlKey) mods.push('Ctrl');
            if (e.altKey) mods.push('Alt');
            if (e.shiftKey) mods.push('Shift');
            if (e.metaKey) mods.push('Win');

            const isModKey = ['Control', 'Alt', 'Shift', 'Meta', 'AltGraph', 'OS'].includes(e.key);

            // Escape with no modifiers cancels the dialog
            if (e.key === 'Escape' && mods.length === 0) {
                close();
                return;
            }

            if (isModKey) {
                // Only modifiers held – show live preview (no main key yet)
                this._renderHkCanvas(mods, false);
            } else {
                const mainKey = this._mapHkKey(e.key);
                if (mainKey && mods.length > 0) {
                    // Valid combo: at least one modifier + main key
                    this._hkPending = [...mods, mainKey].join('+');
                    saveBtn.disabled = false;
                    this._renderHkCanvas([...mods, mainKey], true);
                } else if (mainKey) {
                    // No modifier – show but mark invalid (no save)
                    this._renderHkCanvas([mainKey], false);
                }
            }
        };

        this._hkKeyupFn = (e: KeyboardEvent) => {
            e.preventDefault();
            // After releasing, keep showing the last confirmed combo (or clear live mods)
            if (this._hkPending) {
                this._renderHkCanvas(this._hkPending.split('+'), true);
            } else {
                const mods: string[] = [];
                if (e.ctrlKey) mods.push('Ctrl');
                if (e.altKey) mods.push('Alt');
                if (e.shiftKey) mods.push('Shift');
                if (e.metaKey) mods.push('Win');
                this._renderHkCanvas(mods.length > 0 ? mods : null, false);
            }
        };

        document.addEventListener('keydown', this._hkKeydownFn, true);
        document.addEventListener('keyup', this._hkKeyupFn, true);
    }

    /** Render key chips in the hotkey canvas.
     *  parts=null → show placeholder.
     *  hasMain=true → last chip styled as the main key (accent), rest as modifiers.
     *  hasMain=false → all chips muted (live modifier preview, not yet a valid combo).
     */
    private _renderHkCanvas(parts: string[] | null, hasMain: boolean): void {
        const placeholder = document.getElementById('hotkey-placeholder')!;
        const chipsRow = document.getElementById('hotkey-chips-row') as HTMLElement;

        if (!parts || parts.length === 0) {
            placeholder.style.display = '';
            chipsRow.style.display = 'none';
            chipsRow.innerHTML = '';
            return;
        }

        placeholder.style.display = 'none';
        chipsRow.style.display = 'flex';
        chipsRow.style.opacity = hasMain ? '1' : '0.45';

        const html = parts
            .map((key, i) => {
                const isMain = hasMain && i === parts.length - 1;
                const cls = isMain ? 'hotkey-chip hotkey-chip-main' : 'hotkey-chip';
                const sep = i < parts.length - 1 ? '<div class="hotkey-plus">+</div>' : '';
                return `<div class="${cls}">${escapeHtml(key)}</div>${sep}`;
            })
            .join('');

        chipsRow.innerHTML = html;
    }

    /** Map a browser KeyboardEvent.key value to the string format ParseHotkey expects. */
    private _mapHkKey(key: string): string {
        if (key === ' ') return 'Space';
        if (key === 'Tab') return 'Tab';
        if (key === 'Enter') return 'Enter';
        if (key === 'Backspace') return 'Backspace';
        if (key === 'Delete') return 'Delete';
        if (key === 'Escape') return 'Escape';
        if (/^F([1-9]|1[0-2])$/.test(key)) return key;
        if (/^[a-zA-Z]$/.test(key)) return key.toUpperCase();
        if (/^[0-9]$/.test(key)) return key;
        return '';
    }

    private _bindTabKeyNav(): void {
        const nav = document.querySelector<HTMLElement>('.settings-nav');
        if (!nav) return;
        nav.addEventListener('keydown', (e) => {
            const items = Array.from(nav.querySelectorAll<HTMLElement>('.settings-nav-item'));
            const current = items.findIndex((b) => b.classList.contains('active'));
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                items[(current + 1) % items.length]?.click();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                items[(current - 1 + items.length) % items.length]?.click();
            }
        });
        document.querySelectorAll<HTMLElement>('.settings-nav-item').forEach((btn) => {
            if (!btn.getAttribute('tabindex')) btn.setAttribute('tabindex', '0');
        });
    }
}
