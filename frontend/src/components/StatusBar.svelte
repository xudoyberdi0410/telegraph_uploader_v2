<script>
    export let statusMsg;
</script>

{#if statusMsg}
    <div class="status-bar" class:error={statusMsg.startsWith("Ошибка")}>
        {statusMsg}
    </div>
{/if}

<style>
    .status-bar {
        background: #2a2a2a;
        padding: 5px;
        color: #aaa;
        text-align: center;
        border-bottom: 1px solid #444;
        font-size: 0.8rem;
    }
    .status-bar.error {
        background: #3a1b1b;
        color: #ff6b6b;
    }
</style>
