<script>
    import Header from "../components/Header.svelte";
    import SuccessBox from "../components/SuccessBox.svelte";
    import StatusBar from "../components/StatusBar.svelte";
    import ImageGrid from "../components/ImageGrid.svelte";
    import Footer from "../components/Footer.svelte";
    

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
    }
</script>

<main>
    <Header
        bind:chapterTitle={$chapterTitle}
        isProcessing={$isProcessing}
        hasImages={$images.length > 0}
        on:selectFolder={selectFolderAction}
        on:selectFiles={selectFilesAction}
    />

    <SuccessBox finalUrl={$finalUrl} {copyLink} />

    <StatusBar statusMsg={$statusMsg} />

    <ImageGrid isProcessing={$isProcessing} />
    <Footer
        isProcessing={$isProcessing}
        hasImages={$images.length > 0}
        pageCount={$images.length}
        on:create={createArticleAction}
        on:clear={confirmClear}
    />
</main>

<style>
    main {
        display: flex;
        flex-direction: column;
        height: 100vh;
        position: relative;
    }
</style>
