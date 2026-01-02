<script>
    import { onMount } from "svelte";
    import { OnFileDrop } from "../wailsjs/runtime/runtime";

    // Компоненты
    import Header from "./components/Header.svelte";
    import SuccessBox from "./components/SuccessBox.svelte";
    import StatusBar from "./components/StatusBar.svelte";
    import ImageGrid from "./components/ImageGrid.svelte";

    // Наш новый Store (импортируем переменные и функции)
    import {
        images,
        chapterTitle,
        isProcessing,
        statusMsg,
        finalUrl,
        addImagesFromPaths,
        clearAll,
        createArticleAction,
        selectFilesAction,
        selectFolderAction,
    } from "./stores/appStore.js";

    // --- Инициализация ---
    onMount(() => {
        window.ondragover = function (e) {
            e.preventDefault(); // Это ОБЯЗАТЕЛЬНО, чтобы сработал Drop
            // Можно визуально подсветить окно, если нужно
        };

        window.ondrop = function (e) {
            e.preventDefault(); // Чтобы браузер не пытался открыть файл как картинку
        };
        console.log("OnFileDrop type:", typeof OnFileDrop);
        OnFileDrop((x, y, paths) => {
            console.log("OnFileDrop type:", typeof OnFileDrop);
            console.log("Files dropped from OS:", paths);
            if ($isProcessing) return;
            addImagesFromPaths(paths);
        }, true);
    });

    function confirmClear() {
        if (confirm("Очистить список?")) clearAll();
    }
    function copyLink() {
        navigator.clipboard.writeText($finalUrl);
        statusMsg.set("Ссылка скопирована!");
    }
</script>

<main>
    <Header
        bind:chapterTitle={$chapterTitle}
        isProcessing={$isProcessing}
        hasImages={$images.length > 0}
        on:selectFolder={selectFolderAction}
        on:selectFiles={selectFilesAction}
        on:create={createArticleAction}
        on:clear={confirmClear}
    />

    <SuccessBox finalUrl={$finalUrl} {copyLink} />

    <StatusBar statusMsg={$statusMsg} />

    <ImageGrid isProcessing={$isProcessing} />
</main>

<style>
    :root {
        --bg-color: #1a1a1a;
        --header-bg: #252525;
        --card-bg: #2a2a2a;
        --text-main: #e0e0e0;
        --accent: #4a90e2;
        --border: #333;
        --header-height: 70px;
    }

    :global(body) {
        margin: 0;
        background: var(--bg-color);
        color: var(--text-main);
        font-family: sans-serif;
        overflow: hidden;
        user-select: none;
    }

    main {
        display: flex;
        flex-direction: column;
        height: 100vh;
    }
</style>
