const FILTERS = [
    { label: 'Apps',    value: 'applications' },
    { label: 'Files',   value: 'files' },
    { label: 'Folders', value: 'folders' },
    { label: 'Clip',    value: 'clipboard' },
    { label: 'System',  value: 'system' },
];

export class FilterPills {
    private containerEl: HTMLElement;
    private onChange: (filter: string | null) => void;
    private activeFilter: string | null = null;

    constructor(containerEl: HTMLElement, onChange: (filter: string | null) => void) {
        this.containerEl = containerEl;
        this.onChange = onChange;
    }

    /** Show full pill row (called when query is non-empty). */
    render(activeFilter: string | null): void {
        this.activeFilter = activeFilter;
        this.containerEl.innerHTML = FILTERS.map(f => `
            <button class="filter-pill${activeFilter === f.value ? ' active' : ''}"
                data-filter="${f.value}">${f.label}</button>
        `).join('');
        this.containerEl.classList.add('visible');
        this.containerEl.classList.remove('active-only');
        this._bind();
    }

    /** Show only the active filter as a removable badge (called when query is cleared
     *  but a filter is still set — lets the user know the filter persists). */
    renderActiveOnly(): void {
        const f = FILTERS.find(f => f.value === this.activeFilter);
        if (!f) { this.hide(); return; }
        this.containerEl.innerHTML = `
            <button class="filter-pill active filter-pill-active-badge"
                data-filter="${f.value}"
                title="Clear filter">${f.label} ✕</button>
        `;
        this.containerEl.classList.add('visible', 'active-only');
        this._bind();
    }

    /** Hide the pill row entirely. */
    hide(): void {
        this.containerEl.classList.remove('visible', 'active-only');
    }

    /** Clear the active filter without touching visibility. */
    clearFilter(): void {
        this.activeFilter = null;
    }

    get active(): string | null { return this.activeFilter; }

    private _bind(): void {
        this.containerEl.querySelectorAll<HTMLElement>('.filter-pill').forEach(pill => {
            pill.addEventListener('click', () => {
                const v = pill.dataset['filter'] ?? null;
                const next = this.activeFilter === v ? null : v;
                this.activeFilter = next;
                this.onChange(next);
            });
        });
    }
}
