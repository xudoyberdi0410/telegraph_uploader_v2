<script>
    import { createEventDispatcher } from "svelte";
    const dispatch = createEventDispatcher();
    export let finalUrl;
    export let copyLink;
</script>

{#if finalUrl}
    <div class="success-box">
        <span>✅ Готово:</span>
        <a href={finalUrl} target="_blank">{finalUrl}</a>
        <button class="btn small" on:click={copyLink}>Copy</button>
        <button class="btn tg-btn" on:click={() => dispatch("openTelegram")}>
            ✈️ Отправить в Telegram
        </button>
    </div>
{/if}

<style>
    .success-box {
        background: #1b3a1b;
        color: #4caf50;
        padding: 10px;
        text-align: center;
        border-bottom: 1px solid #2e5c2e;
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 10px;
    }
    .success-box a {
        color: #80e27e;
        text-decoration: none;
        font-weight: bold;
    }
    .tg-btn {
        background: #0088cc;
        color: white;
        margin-left: 10px;
    }
</style>
