import { GetSettings, SaveSettings } from "../../wailsjs/go/main/App";

class SettingsStore {
    settings = $state({
        resize: false,
        resize_to: 1600,
        webp_quality: 80,
        last_channel_id: "0",
        last_channel_hash: "0",
        last_channel_title: ""
    })

    isInitialized = false;
    saveTimer = null;

    constructor() {
        this.loadSettings();
    }

    async loadSettings() {
        try {
            const saved = await GetSettings();
            if (saved) {
                this.settings = saved;
            }
            this.isInitialized = true;
            console.log("Settings loaded:", saved);
        } catch (e) {
            console.error("Failed to load settings:", e);
            this.isInitialized = true;
        }
    }

    triggerAutoSave() {
        if (!this.isInitialized) return;

        clearTimeout(this.saveTimer);
        this.saveTimer = setTimeout(() => {
            const settingsToSave = $state.snapshot(this.settings);
            // Ensure integer for quality
            settingsToSave.webp_quality = Math.round(settingsToSave.webp_quality);
            SaveSettings(settingsToSave);
            console.log("Settings saved");
        }, 500);
    }
}

export const settingsStore = new SettingsStore();
