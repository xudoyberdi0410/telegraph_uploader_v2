import { GetTitles, CreateTitle } from "../../wailsjs/go/main/App";

class TitlesStore {
    titles = $state([]);
    selectedTitleId = $state(0);
    statusMsg = $state("");

    constructor() {
        this.loadTitles();
    }

    async loadTitles() {
        try {
            this.titles = await GetTitles() || [];
        } catch (e) {
            console.error("Failed to load titles:", e);
        }
    }

    async createTitleAction(name, rootFolder) {
        if (!name) return;
        try {
            await CreateTitle(name, rootFolder || "");
            await this.loadTitles();
            // Automatically select the new title
            const newTitle = this.titles.find(t => t.name === name);
            if (newTitle) {
                this.selectedTitleId = newTitle.id;
            }
            this.statusMsg = "Тайтл создан!";
        } catch (e) {
            console.error("Failed to create title:", e);
            this.statusMsg = "Ошибка создания тайтла: " + e;
        }
    }
}

export const titlesStore = new TitlesStore();
