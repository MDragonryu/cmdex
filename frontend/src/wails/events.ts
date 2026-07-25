import { toast } from 'sonner';

export const eventNames = {
    openSettings: 'open-settings',
    openShortcuts: 'open-shortcuts',
    settingsChanged: 'settings-changed',
    settingsWindowClosing: 'settings-window-closing',
    launcherShown: 'launcher-shown',
    launcherHidden: 'launcher-hidden',
};

export async function initEventNames(): Promise<void> {
    try {
        const { GetEventNames } = await import('../../bindings/cmdex/eventservice');
        const names = await GetEventNames();
        eventNames.openSettings = names.openSettings;
        eventNames.openShortcuts = names.openShortcuts;
        eventNames.settingsChanged = names.settingsChanged;
        eventNames.settingsWindowClosing = names.settingsWindowClosing;
        eventNames.launcherShown = names.launcherShown;
        eventNames.launcherHidden = names.launcherHidden;
    } catch (err) {
        console.error('Failed to init event names:', err);
        toast.error('Failed to initialize events. Using fallback event names.');
    }
}