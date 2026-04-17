import { EvalCalc } from '../../wailsjs/go/main/App';

export class CalcPreview {
    private el: HTMLElement;

    constructor(el: HTMLElement) {
        this.el = el;
    }

    async update(query: string): Promise<void> {
        try {
            const result = await EvalCalc(query);
            if (result) {
                this.el.textContent = '= ' + result;
                this.el.setAttribute('aria-hidden', 'false');
                return;
            }
        } catch {
            /* non-critical */
        }
        this.clear();
    }

    clear(): void {
        this.el.textContent = '';
        this.el.setAttribute('aria-hidden', 'true');
    }
}
