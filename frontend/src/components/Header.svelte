<script>
    import { createEventDispatcher } from 'svelte';
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
            bind:value={chapterTitle} 
            placeholder="Название главы (например: Глава 1)"
            disabled={isProcessing}
        />
    </div>

    <div class="button-group">
        <button class="btn btn-secondary" on:click={() => dispatch('selectFolder')} disabled={isProcessing}>
            📁 Выбрать папку
        </button>
        <button class="btn btn-secondary" on:click={() => dispatch('selectFiles')} disabled={isProcessing}>
            🖼️ Выбрать файлы
        </button>
        <button class="btn btn-primary" on:click={() => dispatch('create')} disabled={isProcessing || !hasImages}>
            🚀 Создать страницу
        </button>
        <button class="btn btn-danger" on:click={() => dispatch('clear')} disabled={isProcessing || !hasImages}>
            🗑️ Очистить
        </button>
    </div>
</header>

<style>
    header {
        display: flex;
        flex-direction: column;
        gap: 15px;
        margin-bottom: 20px;
        padding: 20px;
        background: #252525;
        border-radius: 8px;
    }
    .input-group input {
        width: 100%;
        padding: 10px;
        background: #333;
        border: 1px solid #444;
        color: white;
        border-radius: 4px;
    }
    .button-group {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }
</style>
