<script>
    import { createEventDispatcher } from "svelte";
    const dispatch = createEventDispatcher();

    // Входные параметры (props)
    export let chapterTitle = "";
    export let isProcessing = false;
    export let hasImages = false;
</script>

<header>
    <div class="input-group">
        <input
            type="text"
            class="title-input"
            bind:value={chapterTitle}
            placeholder="Название главы (например: Глава 1)"
            disabled={isProcessing}
        />
    </div>

    <div class="button-group">
        <button
            class="btn btn-secondary"
            on:click={() => dispatch("selectFolder")}
            disabled={isProcessing}
        >
            📁 Выбрать папку
        </button>
        <button
            class="btn secondary"
            on:click={() => dispatch("selectFiles")}
            disabled={isProcessing}
        >
            🖼️ Выбрать файлы
        </button>
        <button
            class="btn primary"
            on:click={() => dispatch("create")}
            disabled={isProcessing || !hasImages}
        >
            🚀 Создать страницу
        </button>
        <button
            class="btn danger"
            on:click={() => dispatch("clear")}
            disabled={isProcessing || !hasImages}
        >
            🗑️ Очистить
        </button>
    </div>
</header>

<style>
    header {
        background: var(--header-bg);
        padding: 0 1.5rem;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom: 1px solid var(--border);
        height: var(--header-height);
        flex-shrink: 0;
        gap: 20px;
    }
    .input-group input {
        width: 100%;
        padding: 10px;
        background: #333;
        border: 1px solid #444;
        color: white;
        border-radius: 4px;
    }
    .title-input {
        background: #111;
        border: 1px solid #444;
        color: white;
        padding: 8px 12px;
        border-radius: 4px;
        font-size: 1rem;
        width: 100%;
        max-width: 350px;
    }
    .title-input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .button-group {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }
    .btn {
        padding: 0.6rem 1.2rem;
        border-radius: 6px;
        border: none;
        cursor: pointer;
        font-weight: bold;
        font-size: 0.9rem;
        transition: 0.2s;
    }
    .btn.primary {
        background: var(--accent);
        color: white;
    }
    .btn.primary:hover {
        background: #357abd;
    }
    .btn.primary:disabled {
        background: #555;
        cursor: not-allowed;
    }

    .btn.secondary {
        background: #333;
        color: #ddd;
        border: 1px solid #444;
    }
    .btn.secondary:hover {
        background: #444;
    }
</style>
