<script>
    import { onMount } from "svelte";
    import { OnFileDrop } from "../wailsjs/runtime/runtime";
    
    import Header from "./components/Header.svelte";
    import SuccessBox from "./components/SuccessBox.svelte";
    import StatusBar from "./components/StatusBar.svelte";
    import ImageGrid from "./components/ImageGrid.svelte";

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

    onMount(() => {
        // ВАЖНО: убираем window.ondragover и window.ondrop
        // Они блокируют работу Wails OnFileDrop!
        
        console.log("Initializing OnFileDrop...");
        
        // Wails OnFileDrop работает на уровне нативного окна
        OnFileDrop((x, y, paths) => {
            console.log("Files dropped:", paths);
            if ($isProcessing) {
                console.log("Processing in progress, ignoring drop");
                return;
            }
            addImagesFromPaths(paths);
        }, true); // true = включаем обработчик
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
