<script>
    import { Snackbar, snackbar } from "m3-svelte";

    import { appState } from "../stores/appStore.svelte";

    import Header from "../components/Header.svelte";
    import ImageGrid from "../components/ImageGrid.svelte";
    import Footer from "../components/Footer.svelte";

    function confirmClear() {
        if (confirm("Очистить список?")) appState.clearAll();
    }

    function copyLink() {
        if (appState.finalUrl) {
            navigator.clipboard.writeText(appState.finalUrl);
            appState.statusMsg = "Ссылка скопирована!";
            snackbar("Ссылка скопирована!", undefined, true);
        }
    }

    $effect(() => {
        if (appState.statusMsg) {
            snackbar(appState.statusMsg);
        }
    });

    function handleDragOver(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "copy";
    }

    function handleDrop(e) {
        e.preventDefault();
        const files = Array.from(e.dataTransfer.files);
        if (files.length > 0) {
            const paths = files
                .map((f) => {
                    // @ts-ignore
                    return f.path || f.name;
                })
                .filter((p) => p);

            appState.addImagesFromPaths(paths);
        }
    }
</script>

<main role="application" ondragover={handleDragOver} ondrop={handleDrop}>
    <Header
        bind:chapterTitle={appState.chapterTitle}
        isProcessing={appState.isProcessing}
        onSelectFolder={() => appState.selectFolderAction()}
        onSelectFiles={() => appState.selectFilesAction()}
    />

    <ImageGrid isProcessing={appState.isProcessing} />
    <Footer
        isProcessing={appState.isProcessing}
        hasImages={appState.images.length > 0}
        pageCount={appState.images.length}
        createArticleAction={appState.createArticleAction}
        clearAll={confirmClear}
        {copyLink}
    />
    <Snackbar />
</main>

<style>
    main {
        display: flex;
        flex-direction: column;
        height: 100vh;
        position: relative;
        padding-top: 16px;
    }
</style>
