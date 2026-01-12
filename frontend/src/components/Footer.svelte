<script>
    import iconDelete from "@ktibow/iconset-material-symbols/delete-outline";
    import iconCancel from "@ktibow/iconset-material-symbols/cancel-sharp";
    import iconOpen from "@ktibow/iconset-material-symbols/open-in-new";
    import iconCopy from "@ktibow/iconset-material-symbols/content-copy-outline";
    import iconShare from "@ktibow/iconset-material-symbols/share-outline";
    import { Button, Card, FAB, Icon } from "m3-svelte";
    import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

    import { editorStore } from "../stores/editor.svelte";
    import { navigationStore } from "../stores/navigation.svelte";

    import WavyLinearProgressAnimated from "../ui/WavyLinearProgressAnimated.svelte";

    let {
        isProcessing = false,
        hasImages = false,
        pageCount = 0,
        copyLink = () => {},
        clearAll = () => {},
        createArticleAction = () => {},
    } = $props();

    function publishToTelegram() {
        navigationStore.navigateTo("telegram", {
            historyId: editorStore.currentHistoryId,
            articleTitle: editorStore.chapterTitle,
            titleId: editorStore.currentTitleId || 0,
        });
    }
</script>

<footer>
    {#if isProcessing}
        <div class="card-wrapper">
            <Card variant="filled">
                <WavyLinearProgressAnimated percent={editorStore.uploadProgress > 0 ? editorStore.uploadProgress : 0} />
            </Card>
        </div>
        <Button size="m" square disabled>
            <Icon icon={iconCancel} />
        </Button>
    {:else}
        <Button
            size="m"
            square
            onclick={clearAll}
            disabled={isProcessing || !hasImages}
        >
            <Icon icon={iconDelete} />
        </Button>
        <div class="card-wrapper">
            {#if editorStore.finalUrl}
                <div class="success-container">
                    <FAB
                        text={editorStore.finalUrl}
                        color="tertiary"
                        onclick={copyLink}
                    />
                    <div class="actions">
                        <Button
                            variant="tonal"
                            size="m"
                            square
                            onclick={() => {
                                BrowserOpenURL(editorStore.finalUrl);
                            }}
                        >
                            <Icon icon={iconOpen} />
                        </Button>
                        <Button
                            variant="tonal"
                            onclick={copyLink}
                            size="m"
                            square
                        >
                            <Icon icon={iconCopy} />
                        </Button>
                        <Button
                            variant="tonal"
                            onclick={publishToTelegram}
                            size="m"
                            square
                        >
                            <Icon icon={iconShare} />
                        </Button>
                    </div>
                </div>
            {:else}
                <Card variant="outlined">
                    Pages: {pageCount}
                </Card>
            {/if}
        </div>

        <!-- Only for testing visual feedback, added tonal variant if editing -->
        <Button
            size="m"
            square
            onclick={createArticleAction}
            disabled={isProcessing ||
                !hasImages ||
                (editorStore.editMode && !editorStore.isDirty)}
            variant={editorStore.editMode ? "filled" : "tonal"}
        >
            {editorStore.editMode ? "Обновить" : "Опубликовать"}
        </Button>
    {/if}
</footer>

<style>
    footer {
        position: absolute;
        bottom: 0;
        width: 100%;
        text-align: center;
        padding: 1rem 0;
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 1rem;
    }
    .card-wrapper {
        flex-grow: 1;
        text-align: left;
    }
    .success-container {
        width: 100%;
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }
    .actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }
</style>
