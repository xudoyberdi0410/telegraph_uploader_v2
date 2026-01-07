<script>
    import Header from "../components/Header.svelte";
    import ImageGrid from "../components/ImageGrid.svelte";
    import Footer from "../components/Footer.svelte";
    
    import { Snackbar, snackbar, Button } from "m3-svelte";

    import {
        images,
        chapterTitle,
        isProcessing,
        statusMsg,
        finalUrl,
        clearAll,
        createArticleAction,
        selectFilesAction,
        selectFolderAction,
    } from "../stores/appStore.js";

    function confirmClear() {
        if (confirm("Очистить список?")) clearAll();
    }

    function copyLink() {
        navigator.clipboard.writeText($finalUrl);
        statusMsg.set("Ссылка скопирована!");
        snackbar("Ссылка скопирована!", undefined, true);
    }

    let isSnackbarActive = false;
    let snackbarMessage = "";

    $: if ($statusMsg) {
        snackbarMessage = $statusMsg;
        isSnackbarActive = true;
    }


</script>

<main>
    <Header
        bind:chapterTitle={$chapterTitle}
        isProcessing={$isProcessing}
        on:selectFolder={selectFolderAction}
        on:selectFiles={selectFilesAction}
    />

    <ImageGrid isProcessing={$isProcessing} />
    <Footer
        isProcessing={$isProcessing}
        hasImages={$images.length > 0}
        pageCount={$images.length}

        {createArticleAction}
        clearAll={confirmClear}
        {copyLink}
    />
    <Snackbar/>
</main>

<style>
    main {
        display: flex;
        flex-direction: column;
        height: 100vh;
        position: relative;
    }
</style>
