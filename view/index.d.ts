declare global {
  interface HTMXBeforeSwapEvent extends Event {
    detail: {
      xhr: {
        status: number;
      };
      shouldSwap: boolean;
      isError: boolean;
    };
  }

  interface HTMLElementEventMap {
    ["htmx:beforeSwap"]: HTMXBeforeSwapEvent;
  }

  interface Plaid {
    create: (config: PlaidLinkHandlerConfig) => PlaidHandler
  }

  interface PlaidLinkHandlerConfig {
    token: string;
    onSuccess: (publicToken: string, metadata: unknown) => void | Promise<void>;
    onEvent: (eventName: string, metadata: unknown) => void;
    onExit: (error: unknown, metadata: unknown) => void;
  }

  interface PlaidHandler {
    open: () => void;
    exit: (config?: { force?: boolean }) => void;
    destroy: () => void;
  }

  interface Window {
    Plaid: Plaid;
  }
}

export { };
