import {
    OpenFilesDialog,
    OpenFolderDialog,
    UploadChapter,
    CreateTelegraphPage,
    GetSettings,
    SaveSettings,
    EditTelegraphPage,
    GetTelegraphPage
} from "../../wailsjs/go/main/App"


class AppState {
    images = $state([])
    chapterTitle = $state("")
    isProcessing = $state(false)
    statusMsg = $state("")
    finalUrl = $state("")
    currentPage = $state("home") // Moved from App.svelte

    // Edit Mode State
    editMode = $state(false)
    editArticlePath = $state("")
    editAccessToken = $state("")

    // Change Detection
    savedTitle = $state("")
    savedImagesJson = $state("[]")

    isDirty = $derived(
        this.chapterTitle !== this.savedTitle ||
        JSON.stringify(this.images.map(i => i.id)) !== this.savedImagesJson
    )

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

    // Helper to snapshot current state
    updateSavedState() {
        this.savedTitle = this.chapterTitle;
        this.savedImagesJson = JSON.stringify(this.images.map(i => i.id));
    }

    async loadSettings() {
        try {
            const saved = await GetSettings();
            this.settings = saved;
            this.isInitialized = true;
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
            const settingsToSave = $state.snapshot(this.settings);
            settingsToSave.webp_quality = Math.round(settingsToSave.webp_quality);
            SaveSettings(settingsToSave);
            console.log("Настройки сохранены");
        }, 500);
    }

    addImagesFromPaths(paths) {
        if (!paths || paths.length === 0) return;

        const newImages = paths
            .map((fullPath) => {
                if (!fullPath.match(/\.(jpg|jpeg|png|webp)$/i)) return null;

                const fileName = fullPath.replace(/^.*[\\/]/, "");

                const exists = this.images.find(img => img.originalPath === fullPath);
                if (exists) return null;

                return {
                    id: fullPath,
                    name: fileName,
                    thumbnailSrc: `/thumbnail/${encodeURIComponent(fullPath)}`,
                    originalPath: fullPath,
                    selected: true,
                    type: 'file' // Distinguish from 'url'
                };
            })
            .filter(Boolean);

        if (newImages.length > 0) {
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

        // Reset Edit Mode
        this.editMode = false;
        this.editArticlePath = "";
        this.editAccessToken = "";

        this.updateSavedState(); // Reset dirty state
    }

    async selectFolderAction() {
        try {
            const result = await OpenFolderDialog();
            if (!result || !result.path) return;

            if (!this.editMode) {
                this.chapterTitle = result.title;
                this.images = [];
            }
            if (!this.chapterTitle) {
                this.chapterTitle = result.title;
            }

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

    async loadArticle(historyItem) {
        this.clearAll();
        this.isProcessing = true;
        this.statusMsg = "Загрузка статьи...";

        try {
            // historyItem must have tgph_token (AccessToken)
            // We need it for editing.
            if (!historyItem.tgph_token && !historyItem.TgphToken) {
                // Try to fallback or warn?
                // If token is missing, we might fail to edit if it's not the same as global token.
            }
            const token = historyItem.tgph_token || historyItem.TgphToken || "";

            const pageData = await GetTelegraphPage(historyItem.url);

            this.editMode = true;
            this.editAccessToken = token;
            this.editArticlePath = pageData.path.split('/').pop(); // Extract path from URL just in case
            this.chapterTitle = pageData.title;

            const existingImages = pageData.images.map((url, idx) => ({
                id: url, // URL as ID
                name: `Image ${idx + 1}`,
                thumbnailSrc: url,
                originalPath: url,
                selected: true,
                type: 'url'
            }));

            this.images = existingImages;

            // Set Edit Mode state
            this.editMode = true;
            this.editAccessToken = token;
            this.editArticlePath = pageData.path.split('/').pop();
            this.finalUrl = historyItem.url;

            this.updateSavedState(); // Mark current state as "clean"

            this.statusMsg = "Статья загружена";
            this.currentPage = "home"; // Switch to home
        } catch (e) {
            console.error(e);
            this.statusMsg = "Ошибка загрузки: " + e.message;
            alert("Не удалось загрузить статью: " + e.message);
            this.editMode = false; // Fallback
        } finally {
            this.isProcessing = false;
        }
    }

    createArticleAction = async () => {
        const selectedImages = this.images.filter((img) => img.selected);

        if (selectedImages.length === 0) {
            alert("Список пуст или ничего не выбрано!");
            return;
        }
        if (!this.chapterTitle.trim()) {
            alert("Пожалуйста, введите название главы!");
            return;
        }

        this.isProcessing = true;
        this.finalUrl = "";

        try {
            const settingsSnapshot = $state.snapshot(this.settings);

            // Separate local files from existing URLs
            const localFiles = selectedImages.filter(img => img.type === 'file').map(img => img.originalPath);
            const existingUrls = selectedImages.filter(img => img.type === 'url').map(img => img.originalPath);

            let newLinks = [];
            if (localFiles.length > 0) {
                this.statusMsg = `Загрузка ${localFiles.length} новых изображений...`;
                const uploadRes = await UploadChapter(localFiles, settingsSnapshot);
                if (!uploadRes.success) throw new Error(uploadRes.error);
                newLinks = uploadRes.links;
            }

            // Combine links preserving order? A bit tricky because we separated them.
            // If user reordered images, `selectedImages` is in correct order.
            // We need to map `selectedImages` to final URLs.

            // Optimization: Create a map of localPath -> uploadedUrl
            // But Wait, UploadChapter returns list of links corresponding to input list?
            // "The order of files in the input list is preserved." - usually yes.

            // Let's assume UploadChapter returns links in same order as input 'localFiles'.
            let localFileIndex = 0;
            const finalImageUrls = selectedImages.map(img => {
                if (img.type === 'url') {
                    return img.originalPath;
                } else {
                    const link = newLinks[localFileIndex];
                    localFileIndex++;
                    return link;
                }
            });

            if (this.editMode) {
                this.statusMsg = "Обновление статьи в Telegraph...";
                // Use editArticlePath. If we have full URL, we need slug.
                // loadArticle sets editArticlePath to slug.
                const path = this.editArticlePath;
                const resultUrl = await EditTelegraphPage(path, this.chapterTitle, finalImageUrls, this.editAccessToken);

                if (resultUrl.startsWith("http")) {
                    this.finalUrl = resultUrl;
                    this.statusMsg = "Статья обновлена!";
                    // Update state to match clean
                    this.refreshImagesAfterSave(finalImageUrls);
                } else {
                    throw new Error(resultUrl);
                }

            } else {
                this.statusMsg = "Создание статьи в Telegraph...";
                const telegraphLink = await CreateTelegraphPage(this.chapterTitle, finalImageUrls);

                if (telegraphLink.startsWith("http")) {
                    this.finalUrl = telegraphLink;
                    this.statusMsg = "Готово!";

                    // Switch to Edit Mode for this new article
                    this.editMode = true;
                    // Extract path: https://telegra.ph/Some-Title-01-09 -> Some-Title-01-09
                    const parts = telegraphLink.split('/');
                    this.editArticlePath = parts[parts.length - 1];
                    this.editAccessToken = ""; // Use internal default

                    this.refreshImagesAfterSave(finalImageUrls);
                } else {
                    throw new Error(telegraphLink);
                }
            }

        } catch (e) {
            this.statusMsg = "Ошибка: " + e.message;
        } finally {
            this.isProcessing = false;
        }
    }

    refreshImagesAfterSave(newUrls) {
        // Replace all images with type='url' and new paths, preserving order
        // This ensures next save sees them as existing URLs
        this.images = newUrls.map((url, idx) => ({
            id: url,
            name: `Image ${idx + 1}`, // Or keep original name if possible? Hard since we mapped.
            thumbnailSrc: url,
            originalPath: url,
            selected: true,
            type: 'url'
        }));
        this.updateSavedState();
    }
}

export const appState = new AppState()
