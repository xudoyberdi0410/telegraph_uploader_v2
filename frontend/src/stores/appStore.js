import { writable, get } from 'svelte/store';

import {
    OpenFilesDialog,
    OpenFolderDialog,
    UploadChapter,
    CreateTelegraphPage
} from "../../wailsjs/go/main/App"

// 1. Состояние (переменные)
export const images = writable([]);
export const chapterTitle = writable("");
export const isProcessing = writable(false);
export const statusMsg = writable("");
export const finalUrl = writable("");

// 2. Логика (Actions)

// Добавление картинок (фильтрация и форматирование)
export const addImagesFromPaths = (paths) => {
    if (!paths || paths.length === 0) return;

    const currentImages = get(images); // Получаем текущий список
    
    const newImages = paths
        .map((fullPath) => {
            // Пропускаем не картинки
            if (!fullPath.match(/\.(jpg|jpeg|png|webp)$/i)) return null;

            const safePath = encodeURIComponent(fullPath);
            const fileName = fullPath.replace(/^.*[\\/]/, "");

            // Проверяем, нет ли уже такого файла в списке, чтобы избежать дублей
            const exists = currentImages.find(img => img.originalPath === fullPath);
            if (exists) return null;

            return {
                id: fullPath,
                name: fileName,
                thumbnailSrc: `/thumbnail/${safePath}`,
                originalPath: fullPath,
                selected: true,
            };
        })
        .filter(Boolean); // Убираем null

    if (newImages.length > 0) {
        images.update(n => [...n, ...newImages]);
        statusMsg.set(`Добавлено ${newImages.length} файлов`);
    }
};

// Удаление картинки по индексу
export const removeImageByIndex = (index) => {
    images.update(items => {
        const newItems = [...items];
        newItems.splice(index, 1);
        return newItems;
    });
};

// Очистка всего
export const clearAll = () => {
    images.set([]);
    chapterTitle.set("");
    statusMsg.set("");
    finalUrl.set("");
};

export const selectFolderAction = async () => {
    try {
        const result = await OpenFolderDialog();
        if (!result || !result.path) return;

        chapterTitle.set(result.title);
        images.set([]); // Сброс
        
        // Тут нужно вызвать логику добавления. 
        // Если addImagesFromPaths экспортирована, вызываем её, 
        // либо дублируем логику обновления стора.
        addImagesFromPaths(result.images); 
    } catch (err) {
        console.error(err);
        statusMsg.set("Ошибка выбора папки");
    }
};

export const selectFilesAction = async () => {
    try {
        const files = await OpenFilesDialog();
        if (files && files.length > 0) {
            addImagesFromPaths(files);
        }
    } catch (err) {
        console.error(err);
    }
};

export const createArticleAction = async () => {
    const $images = get(images);
    const $chapterTitle = get(chapterTitle);

    const selectedImages = $images.filter((img) => img.selected);
    if (selectedImages.length === 0) {
        alert("Список пуст или ничего не выбрано!");
        return;
    }
    if (!$chapterTitle.trim()) {
        alert("Пожалуйста, введите название главы!");
        return;
    }

    const filesToUpload = selectedImages.map((img) => img.originalPath);
    
    isProcessing.set(true);
    finalUrl.set("");
    statusMsg.set(`Загрузка ${filesToUpload.length} изображений...`);

    try {
        const uploadRes = await UploadChapter(filesToUpload);
        if (!uploadRes.success) throw new Error(uploadRes.error);

        statusMsg.set("Создание статьи в Telegraph...");
        const telegraphLink = await CreateTelegraphPage($chapterTitle, uploadRes.links);

        if (telegraphLink.startsWith("http")) {
            finalUrl.set(telegraphLink);
            statusMsg.set("Готово!");
        } else {
            throw new Error(telegraphLink);
        }
    } catch (e) {
        statusMsg.set("Ошибка: " + e.message);
    } finally {
        isProcessing.set(false);
    }
};
