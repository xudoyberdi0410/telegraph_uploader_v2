<script>
    import { Snackbar, snackbar } from "m3-svelte";

    import { editorStore } from "../stores/editor.svelte";

    import Header from "../components/Header.svelte";
    import ImageGrid from "../components/ImageGrid.svelte";
    import Footer from "../components/Footer.svelte";

    function confirmClear() {
        if (confirm("Очистить список?")) editorStore.clearAll();
    }

    function copyLink() {
        if (editorStore.finalUrl) {
            navigator.clipboard.writeText(editorStore.finalUrl);
            editorStore.statusMsg = "Ссылка скопирована!";
            snackbar("Ссылка скопирована!", undefined, true);
        }
    }

    $effect(() => {
        if (editorStore.statusMsg) {
            snackbar(editorStore.statusMsg);
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

            editorStore.addImagesFromPaths(paths);
        }
    }
</script>

<main role="application" ondragover={handleDragOver} ondrop={handleDrop}>
    <Header
        bind:chapterTitle={editorStore.chapterTitle}
        isProcessing={editorStore.isProcessing}
        onSelectFolder={() => editorStore.selectFolderAction()}
        onSelectFiles={() => editorStore.selectFilesAction()}
    />

    <ImageGrid isProcessing={editorStore.isProcessing} />
    <Footer
        isProcessing={editorStore.isProcessing}
        hasImages={editorStore.images.length > 0}
        pageCount={editorStore.images.length}
        createArticleAction={editorStore.createArticleAction}
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
