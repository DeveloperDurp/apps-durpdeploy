import Alpine from 'alpinejs'
import htmx from 'htmx.org'

window.Alpine = Alpine
window.htmx = htmx

Alpine.data('toast', () => ({
	visible: false,
	message: '',
	type: 'success',
	timeout: null,
	show(msg, type = 'success') {
		if (this.timeout) clearTimeout(this.timeout);
		this.message = String(msg);
		this.type = type;
		this.visible = true;
		this.timeout = setTimeout(() => {
			this.visible = false;
		}, 3000);
	},
	get alertClass() {
		return this.type === 'error' ? 'alert-error' : 'alert-success';
	},
	init() {
		window.addEventListener('show-toast', (e) => {
			const { message, type } = e.detail;
			this.show(message, type);
		});
		
		document.body.addEventListener('htmx:afterRequest', (e) => {
			const trigger = e.detail.elt;
			const successMsg = trigger.getAttribute('data-toast-success');
			const errorMsg = trigger.getAttribute('data-toast-error');
			
			if (e.detail.successful && successMsg) {
				this.show(successMsg, 'success');
			} else if (!e.detail.successful && errorMsg) {
				this.show(errorMsg, 'error');
			}
		});
	}
}));

window.showToast = (msg, type = 'success') => {
	window.dispatchEvent(new CustomEvent('show-toast', { detail: { message: msg, type } }));
};

Alpine.start()
