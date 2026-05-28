import { toast } from 'sonner';

export const eventNames = {
    openSettings: 'open-settings',
    openShortcuts: 'open-shortcuts',
    settingsChanged: 'settings-changed',
    settingsWindowClosing: 'settings-window-closing',
    ptyOutput: 'pty-output',
    ptyExit: 'pty-exit',
    ptyCleared: 'pty-cleared',
    cmdExecuting: 'cmd-executing',
};

export async function initEventNames(): Promise<void> {
    try {
        const { GetEventNames } = await import('../../bindings/cmdex/eventservice');
        const names = await GetEventNames();
        eventNames.openSettings = names.openSettings;
        eventNames.openShortcuts = names.openShortcuts;
        eventNames.settingsChanged = names.settingsChanged;
        eventNames.settingsWindowClosing = names.settingsWindowClosing;
        eventNames.ptyOutput = names.ptyOutput;
        eventNames.ptyExit = names.ptyExit;
        eventNames.ptyCleared = names.ptyCleared;
        eventNames.cmdExecuting = names.cmdExecuting;
    } catch (err) {
        console.error('Failed to init event names:', err);
        toast.error('Failed to initialize events. Using fallback event names.');
    }
}