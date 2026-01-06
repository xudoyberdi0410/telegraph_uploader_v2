<script>
    import iconDelete from "@ktibow/iconset-material-symbols/delete-outline";
    import iconCancel from "@ktibow/iconset-material-symbols/cancel-sharp";
    import iconOpen from "@ktibow/iconset-material-symbols/open-in-new";
    import iconCopy from "@ktibow/iconset-material-symbols/content-copy-outline";
    import { Button, Card, FAB, Icon, Snackbar, WavyLinearProgress } from "m3-svelte";
    import { createEventDispatcher } from "svelte";
    import { finalUrl } from "../stores/appStore";
    const dispatch = createEventDispatcher();
    export let isProcessing = false;
    export let hasImages = false;
    export let pageCount = 0;
    export let copyLink = () => {};
    // finalUrl.set(
    //     "https://telegra.ph/Deti-semi-Siundzi---Glavy-63-Ongoing-JP-01-03-3",
    // );
</script>

<footer>
    
    {#if isProcessing}
        <div class="card-wrapper">
            <Card variant="filled">
                <WavyLinearProgress percent={100} />
            </Card>
        </div>
        <Button
            size="m"
            square
            onclick={() => dispatch("create")}
            disabled={isProcessing || !hasImages}
        >
            <Icon icon={iconCancel} />
        </Button>
    {:else if $finalUrl}
        <div class="success-container">
            <div class="link-info">
                <Button
                    square
                    onclick={() => dispatch("clear")}
                    disabled={isProcessing || !hasImages}
                    size="m"
                >
                    <Icon icon={iconDelete} />
                </Button>
                <FAB
                    text={$finalUrl}
                    color="tertiary"
                    onclick={copyLink}
                />
            </div>

            <div class="actions">
                <Button variant="tonal" size="m" square>
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
            </div>
        </div>
    {:else}
        <Button
            size="m"
            square
            onclick={() => dispatch("clear")}
            disabled={isProcessing || !hasImages}
        >
            <Icon icon={iconDelete} />
        </Button>
        <div class="card-wrapper">
            <Card variant="outlined">
                Pages: {pageCount}
            </Card>
        </div>
        <Button
            size="m"
            square
            onclick={() => dispatch("create")}
            disabled={isProcessing || !hasImages}
        >
            Telegraph</Button
        >
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
    .link-info {
        display: flex;
        align-items: center;
        gap: 8px;
        overflow: hidden;
    }
    .actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }
</style>
