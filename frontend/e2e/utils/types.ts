// Type declaration for the test seed global injected via page.addInitScript.
// The `declare global` augments the global Window interface so the cast is
// unnecessary at call sites in addInitScript callbacks (which are serialized
// to the browser context and cannot close over module-scope helpers).

export interface CmdexE2ESeed {
    categories?: Array<Record<string, unknown>>;
    commands?: Array<Record<string, unknown>>;
    settings?: Record<string, unknown>;
}

declare global {
    interface Window {
        __cmdexE2E_SEED__?: CmdexE2ESeed;
        // Exposed by the runtime mock so tests can drive Wails events
        // (e.g. `settings-changed`, normally emitted by the settings window).
        __cmdexE2E?: {
            reset(): void;
            seed(data: CmdexE2ESeed): void;
            emit(eventName: string, data: unknown): void;
        };
    }
}

export {};
