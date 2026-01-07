import {
    OpenFilesDialog,
    OpenFolderDialog,
    UploadChapter,
    CreateTelegraphPage,
    GetSettings,
    SaveSettings
} from "../../wailsjs/go/main/App"

class AppState {
    images = $state([])
    chapterTitle = $state("")
    isProcessing = $state(false)
    statusMsg = $state("")
    finalUrl = $state("")

    settings = $state({
        resize: false,
        resize_to: 1600,
        webp_quality: 80
    })

    isInitialized = false
    saveTimer = null

    constructor() {
        this.loadSettings()
    }

    async loadSettings() {
        try {
            const saved = await GetSettings();
            this.settings = saved;
            this.isInitialized = true; // Разрешаем сохранение
            console.log("Настройки загружены:", saved);
        } catch (e) {
            console.error("Не удалось загрузить настройки:", e);
            this.isInitialized = true;
        }
    }

    triggerAutoSave() {
        if (!this.isInitialized) return;

        clearTimeout(this.saveTimer)
        this.saveTimer = setTimeout(() => {
            SaveSettings($state.snapshot(this.settings));
            console.log("Настройки сохранены");
        }, 500);
    }

    addImagesFromPaths(paths) {
        if (!paths || paths.length === 0) return;

        const newImages = paths
            .map((fullPath) => {
                if (!fullPath.match(/\.(jpg|jpeg|png|webp)$/i)) return null;

                const fileName = fullPath.replace(/^.*[\\/]/, "");

                // Обращаемся к this.images напрямую
                const exists = this.images.find(img => img.originalPath === fullPath);
                if (exists) return null;

                return {
                    id: fullPath,
                    name: fileName,
                    thumbnailSrc: `/thumbnail/${encodeURIComponent(fullPath)}`,
                    originalPath: fullPath,
                    selected: true,
                };
            })
            .filter(Boolean);

        if (newImages.length > 0) {
            // Мутируем массив напрямую! Svelte 5 это отследит.
            this.images.push(...newImages);
            this.statusMsg = `Добавлено ${newImages.length} файлов`;
        }
    }

    removeImageByIndex(index) {
        this.images.splice(index, 1);
    }

    clearAll() {
        this.images = [];
        this.chapterTitle = "";
        this.statusMsg = "";
        this.finalUrl = "";
    }

    async selectFolderAction() {
        try {
            const result = await OpenFolderDialog();
            if (!result || !result.path) return;

            this.chapterTitle = result.title;
            this.images = []; // Очистка

            // Если images в result это массив путей
            this.addImagesFromPaths(result.images);
        } catch (err) {
            console.error(err);
            this.statusMsg = "Ошибка выбора папки";
        }
    }

    async selectFilesAction() {
        try {
            const files = await OpenFilesDialog();
            if (files && files.length > 0) {
                this.addImagesFromPaths(files);
            }
        } catch (err) {
            console.error(err);
        }
    }

    async createArticleAction() {
        const selectedImages = this.images.filter((img) => img.selected);

        if (selectedImages.length === 0) {
            alert("Список пуст или ничего не выбрано!");
            return;
        }
        if (!this.chapterTitle.trim()) {
            alert("Пожалуйста, введите название главы!");
            return;
        }

        const filesToUpload = selectedImages.map((img) => img.originalPath);

        this.isProcessing = true;
        this.finalUrl = "";
        this.statusMsg = `Загрузка ${filesToUpload.length} изображений...`;

        try {
            // $state.snapshot создает чистый JS объект из прокси Svelte, 
            // это важно при передаче данных в Wails/JSON
            const settingsSnapshot = $state.snapshot(this.settings);

            const uploadRes = await UploadChapter(filesToUpload, settingsSnapshot);
            if (!uploadRes.success) throw new Error(uploadRes.error);

            this.statusMsg = "Создание статьи в Telegraph...";
            const telegraphLink = await CreateTelegraphPage(this.chapterTitle, uploadRes.links);

            if (telegraphLink.startsWith("http")) {
                this.finalUrl = telegraphLink;
                this.statusMsg = "Готово!";
            } else {
                throw new Error(telegraphLink);
            }
        } catch (e) {
            this.statusMsg = "Ошибка: " + e.message;
        } finally {
            this.isProcessing = false;
        }
    }
}

export const appState = new AppState()
