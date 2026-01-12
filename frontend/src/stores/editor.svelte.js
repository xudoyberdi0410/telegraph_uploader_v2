import {
    OpenFilesDialog,
    OpenFolderDialog,
    UploadChapter,
    CreateTelegraphPage,
    EditTelegraphPage,
    GetTelegraphPage
} from "../../wailsjs/go/main/App";

import { settingsStore } from "./settings.svelte.js";
import { titlesStore } from "./titles.svelte.js";
import { navigationStore } from "./navigation.svelte.js";

class EditorStore {
    images = $state([]);
    chapterTitle = $state("");
    isProcessing = $state(false);
    statusMsg = $state("");
    finalUrl = $state("");

    // Edit Mode State
    editMode = $state(false);
    editArticlePath = $state("");
    editAccessToken = $state("");
    currentHistoryId = $state(0);
    currentTitleId = $state(0);

    // Change Detection
    savedTitle = $state("");
    savedImagesJson = $state("[]");

    isDirty = $derived(
        this.chapterTitle !== this.savedTitle ||
        JSON.stringify(this.images.map(i => i.id)) !== this.savedImagesJson
    );

    updateSavedState() {
        this.savedTitle = this.chapterTitle;
        this.savedImagesJson = JSON.stringify(this.images.map(i => i.id));
    }

    detectTitleFromPath(filePath) {
        if (!filePath || !titlesStore.titles || titlesStore.titles.length === 0) return null;

        const normalizedFilePath = filePath.replace(/\\/g, '/').toLowerCase();
        let bestMatch = null;
        let maxLen = 0;

        for (const title of titlesStore.titles) {
            if (!title.folders) continue;
            for (const folder of title.folders) {
                if (!folder.path) continue;
                const normalizedFolderPath = folder.path.replace(/\\/g, '/').toLowerCase();

                if (normalizedFilePath.startsWith(normalizedFolderPath)) {
                    if (normalizedFolderPath.length > maxLen) {
                        maxLen = normalizedFolderPath.length;
                        bestMatch = title;
                    }
                }
            }
        }
        return bestMatch;
    }

    addImagesFromPaths(paths) {
        if (!paths || paths.length === 0) return;

        // Auto-detect title if not selected
        if (titlesStore.selectedTitleId === 0 && paths.length > 0) {
            const detectedTitle = this.detectTitleFromPath(paths[0]);
            if (detectedTitle) {
                titlesStore.selectedTitleId = detectedTitle.id;
                console.log(`Auto-detected title: ${detectedTitle.name}`);
            }
        }

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
                    type: 'file'
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
        this.editMode = false;
        this.editArticlePath = "";
        this.editAccessToken = "";
        this.currentHistoryId = 0;
        this.currentTitleId = 0;
        this.updateSavedState();
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
            this.currentHistoryId = historyItem.id;
            this.currentTitleId = historyItem.title_id || 0;
            const token = historyItem.tgph_token || historyItem.TgphToken || "";

            const pageData = await GetTelegraphPage(historyItem.url);

            this.editMode = true;
            this.editAccessToken = token;
            this.editArticlePath = pageData.path.split('/').pop();
            this.chapterTitle = pageData.title;

            const existingImages = pageData.images.map((url, idx) => ({
                id: url,
                name: `Image ${idx + 1}`,
                thumbnailSrc: url,
                originalPath: url,
                selected: true,
                type: 'url'
            }));

            this.images = existingImages;
            this.finalUrl = historyItem.url;
            this.updateSavedState();

            this.statusMsg = "Статья загружена";
            navigationStore.navigateTo("home");
        } catch (e) {
            console.error(e);
            this.statusMsg = "Ошибка загрузки: " + e.message;
            alert("Не удалось загрузить статью: " + e.message);
            this.editMode = false;
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
            const settingsSnapshot = $state.snapshot(settingsStore.settings);

            const localFiles = selectedImages.filter(img => img.type === 'file').map(img => img.originalPath);
            
            let newLinks = [];
            if (localFiles.length > 0) {
                this.statusMsg = `Загрузка ${localFiles.length} новых изображений...`;
                const uploadRes = await UploadChapter(localFiles, settingsSnapshot);
                if (!uploadRes.success) throw new Error(uploadRes.error);
                newLinks = uploadRes.links;
            }

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
                const path = this.editArticlePath;
                const resultUrl = await EditTelegraphPage(path, this.chapterTitle, finalImageUrls, this.editAccessToken);

                if (resultUrl.startsWith("http")) {
                    this.finalUrl = resultUrl;
                    this.statusMsg = "Статья обновлена!";
                    this.refreshImagesAfterSave(finalImageUrls);
                } else {
                    throw new Error(resultUrl);
                }

            } else {
                this.statusMsg = "Создание статьи в Telegraph...";
                const titleIdToUse = titlesStore.selectedTitleId ? titlesStore.selectedTitleId : 0;
                const response = await CreateTelegraphPage(this.chapterTitle, finalImageUrls, titleIdToUse);

                if (response.success) {
                    this.finalUrl = response.url;
                    this.currentHistoryId = response.history_id;
                    this.statusMsg = "Готово!";

                    this.editMode = true;
                    const parts = response.url.split('/');
                    this.editArticlePath = parts[parts.length - 1];
                    this.editAccessToken = "";
                    this.currentTitleId = titleIdToUse;

                    this.refreshImagesAfterSave(finalImageUrls);
                } else {
                    throw new Error(response.error || "Unknown error creating page");
                }
            }

        } catch (e) {
            this.statusMsg = "Ошибка: " + e.message;
        } finally {
            this.isProcessing = false;
        }
    }

    refreshImagesAfterSave(newUrls) {
        this.images = newUrls.map((url, idx) => ({
            id: url,
            name: `Image ${idx + 1}`,
            thumbnailSrc: url,
            originalPath: url,
            selected: true,
            type: 'url'
        }));
        this.updateSavedState();
    }
}

export const editorStore = new EditorStore();
