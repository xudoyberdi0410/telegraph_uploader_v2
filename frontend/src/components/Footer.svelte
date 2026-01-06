<script>
    import iconDelete from "@ktibow/iconset-material-symbols/delete-outline";
    import { Button, Card, Icon } from "m3-svelte";
    import { createEventDispatcher } from "svelte";
    const dispatch = createEventDispatcher();
    export let isProcessing = false;
    export let hasImages = false;
    export let pageCount = 0;
</script>

<footer>
    <Button
        size="m"
        square
        onclick={() => dispatch("clear")}
        disabled={isProcessing || !hasImages}
    >
        <Icon icon={iconDelete} />
    </Button>
    <div class="card-wrapper">
        <Card variant="outlined">Pages: {pageCount}</Card>
    </div>
    <Button
        size="m"
        square
        onclick={() => dispatch("create")}
        disabled={isProcessing || !hasImages}>Telegraph</Button
    >
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
        align-items: stretch;
        gap: 1rem;
    }
    .card-wrapper {
        flex-grow: 1;
        text-align: left;
    }
</style>
