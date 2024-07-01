
declare global {
	interface HTMXBeforeSwapEvent extends Event {
		detail: {
			xhr: {
				status: number;
			},
			shouldSwap: boolean;
			isError: boolean;
		}
	}

	interface HTMLElementEventMap {
		["htmx:beforeSwap"]: HTMXBeforeSwapEvent
	}
}

export { }
