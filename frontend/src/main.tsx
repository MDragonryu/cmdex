import React, { useState, useEffect, useCallback, useRef } from 'react'
import {createRoot} from 'react-dom/client'
import './i18n'
import './style.css'
import App from './App'
import SettingsPage from './components/SettingsPage'
import Launcher from './components/Launcher'
import { GetSettings, SetSettings } from '../bindings/cmdex/settingsservice'
import { THEMES, type CustomTheme } from './types'
import { Events } from '@wailsio/runtime'
import { eventNames } from './wails/events'
import { toast } from 'sonner'
import { applyTheme, applyDensity, applyFonts } from './lib/theme-apply'

const container = document.getElementById('root')

const windowKind = new URLSearchParams(window.location.search).get('window')
const isSettingsWindow = windowKind === 'settings'
const isLauncherWindow = windowKind === 'launcher'

const root = createRoot(container!)

if (isLauncherWindow) {
    // The global quick launcher window. It only needs theming applied — all of
    // its behaviour lives in <Launcher />, which stays mounted for the lifetime
    // of the app because the Go side shows/hides the window rather than
    // creating and destroying it.
    function LauncherWindow() {
        const [theme, setTheme] = useState('vscode-dark')

        const applySettings = useCallback((s: Awaited<ReturnType<typeof GetSettings>> | null) => {
            if (!s) return
            const t = s.theme || 'vscode-dark'
            let custom: CustomTheme | undefined
            if (s.customThemes && s.customThemes !== '[]') {
                try {
                    const parsed = JSON.parse(s.customThemes)
                    if (Array.isArray(parsed)) custom = parsed.find((c: CustomTheme) => c.id === t)
                } catch { /* ignore parse error */ }
            }
            setTheme(t)
            applyTheme(t, custom?.colors ?? null)
            applyDensity(s.density || 'comfortable')
            applyFonts(s.uiFont || 'Inter', s.monoFont || 'JetBrains Mono')
        }, [])

        useEffect(() => {
            GetSettings().then(applySettings).catch(() => {})
        }, [applySettings])

        // Stay in sync when preferences change in the settings window.
        useEffect(() => {
            const cleanup = Events.On(eventNames.settingsChanged, () => {
                GetSettings().then(applySettings).catch(() => {})
            })
            return () => cleanup()
        }, [applySettings])

        return <Launcher theme={theme} />
    }

    root.render(
        <React.StrictMode>
            <LauncherWindow />
        </React.StrictMode>
    )
} else if (isSettingsWindow) {
    function SettingsWindow() {
        const [theme, setTheme] = useState('vscode-dark')
        const [density, setDensity] = useState('comfortable')
        const [uiFont, setUiFont] = useState('Inter')
        const [monoFont, setMonoFont] = useState('JetBrains Mono')
        const [customThemes, setCustomThemes] = useState<CustomTheme[]>([])
        const [locale, setLocale] = useState('en')
        const [terminal, setTerminal] = useState('')
        const [lastDarkTheme, setLastDarkTheme] = useState('vscode-dark')
        const [lastLightTheme, setLastLightTheme] = useState('vscode-light')
        const [windowX, setWindowX] = useState(-1)
        const [windowY, setWindowY] = useState(-1)
        const [windowWidth, setWindowWidth] = useState(640)
        const [windowHeight, setWindowHeight] = useState(520)
        const customThemesStrRef = useRef('[]')
        // Mirrors `customThemes` so handlers that run before the next render
        // (import -> onThemeChange in the same tick) still see the latest list.
        const customThemesRef = useRef<CustomTheme[]>([])

        const syncCustomThemes = useCallback((list: CustomTheme[]) => {
            customThemesRef.current = list
            customThemesStrRef.current = JSON.stringify(list)
            setCustomThemes(list)
        }, [])

        useEffect(() => {
            GetSettings().then(s => {
                if (!s) return
                const t = s.theme || 'vscode-dark'
                setTheme(t)
                setDensity(s.density || 'comfortable')
                setUiFont(s.uiFont || 'Inter')
                setMonoFont(s.monoFont || 'JetBrains Mono')
                setLocale(s.locale || 'en')
                setTerminal(s.terminal || '')
                if (s.windowX !== undefined) setWindowX(s.windowX)
                if (s.windowY !== undefined) setWindowY(s.windowY)
                if (s.windowWidth !== undefined) setWindowWidth(s.windowWidth)
                if (s.windowHeight !== undefined) setWindowHeight(s.windowHeight)
                // Parse custom themes before the first applyTheme so a saved custom
                // theme paints with its own colors instead of the default palette.
                let loadedCustomThemes: CustomTheme[] = []
                if (s.customThemes && s.customThemes !== '[]') {
                    try {
                        const parsed = JSON.parse(s.customThemes)
                        loadedCustomThemes = Array.isArray(parsed) ? parsed : []
                        syncCustomThemes(loadedCustomThemes)
                    } catch { /* ignore parse error */ }
                }
                const loadedCustom = loadedCustomThemes.find(c => c.id === t)
                applyTheme(t, loadedCustom?.colors ?? null)
                applyDensity(s.density || 'comfortable')
                applyFonts(s.uiFont || 'Inter', s.monoFont || 'JetBrains Mono')
            }).catch(() => {})
        }, [syncCustomThemes])

        const persistSettings = useCallback(async (newSettings: Record<string, unknown>) => {
            try {
                await SetSettings(JSON.stringify(newSettings))
                Events.Emit(eventNames.settingsChanged, newSettings)
            } catch (err) {
                toast.error('Failed to save settings: ' + String(err))
                console.error('SetSettings error:', err)
            }
        }, [])

        const handleThemeChange = useCallback((newTheme: string) => {
            const builtIn = THEMES.find(t => t.id === newTheme)
            const custom = customThemesRef.current.find(t => t.id === newTheme)
            const themeType = builtIn?.type ?? custom?.type ?? 'dark'
            applyTheme(newTheme, custom?.colors ?? null)
            if (themeType === 'dark') {
                setLastDarkTheme(newTheme)
                document.documentElement.style.setProperty('--cmdex-last-dark-theme', newTheme)
            } else {
                setLastLightTheme(newTheme)
                document.documentElement.style.setProperty('--cmdex-last-light-theme', newTheme)
            }
            setTheme(newTheme)
            const newSettings = {
                locale, terminal, theme: newTheme,
                lastDarkTheme: themeType === 'dark' ? newTheme : lastDarkTheme,
                lastLightTheme: themeType === 'light' ? newTheme : lastLightTheme,
                customThemes: customThemesStrRef.current,
                uiFont, monoFont, density,
                windowX, windowY, windowWidth, windowHeight,
            }
            persistSettings(newSettings)
        }, [locale, terminal, uiFont, monoFont, density, persistSettings, lastDarkTheme, lastLightTheme, windowX, windowY, windowWidth, windowHeight])

        const handleImportTheme = useCallback((newTheme: CustomTheme) => {
            const updated = [...customThemesRef.current, newTheme]
            syncCustomThemes(updated)
            const newSettings = {
                locale, terminal, theme, lastDarkTheme, lastLightTheme,
                customThemes: customThemesStrRef.current, uiFont, monoFont, density,
                windowX, windowY, windowWidth, windowHeight,
            }
            persistSettings(newSettings)
        }, [syncCustomThemes, locale, terminal, theme, uiFont, monoFont, density, persistSettings, lastDarkTheme, lastLightTheme, windowX, windowY, windowWidth, windowHeight])

        const handleRemoveCustomTheme = useCallback((themeId: string) => {
            const updated = customThemesRef.current.filter(t => t.id !== themeId)
            syncCustomThemes(updated)
            const newSettings = {
                locale, terminal, theme, lastDarkTheme, lastLightTheme,
                customThemes: customThemesStrRef.current, uiFont, monoFont, density,
                windowX, windowY, windowWidth, windowHeight,
            }
            persistSettings(newSettings)
            if (theme === themeId) {
                handleThemeChange('vscode-dark')
            }
        }, [syncCustomThemes, locale, terminal, theme, uiFont, monoFont, density, persistSettings, lastDarkTheme, lastLightTheme, windowX, windowY, windowWidth, windowHeight, handleThemeChange])

        return (
            <SettingsPage
                theme={theme}
                onThemeChange={handleThemeChange}
                customThemes={customThemes}
                onImportTheme={handleImportTheme}
                onRemoveCustomTheme={handleRemoveCustomTheme}
                density={density}
                uiFont={uiFont}
                monoFont={monoFont}
            />
        )
    }

    root.render(
        <React.StrictMode>
            <SettingsWindow />
        </React.StrictMode>
    )
} else {
    root.render(
        <React.StrictMode>
            <App/>
        </React.StrictMode>
    )
}