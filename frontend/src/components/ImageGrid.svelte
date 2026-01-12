<script>
    import ImageCard from "./ImageCard.svelte";

    import { editorStore } from "../stores/editor.svelte";

    let { isProcessing } = $props();

    let draggedIndex = $state(null);

    function handleDragStart(e, index) {
        draggedIndex = index;
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.dropEffect = "move";
        e.target.style.opacity = "0.5";
    }
    function handleDragOver(e, index) {
        e.preventDefault();
        if (draggedIndex === null || draggedIndex === index) return;

        const sourceIdx = draggedIndex;
        const targetIdx = index;

        const item = editorStore.images[sourceIdx];
        editorStore.images.splice(sourceIdx, 1);
        editorStore.images.splice(targetIdx, 0, item);

        draggedIndex = targetIdx;
    }

    function handleDragEnd(e) {
        draggedIndex = null;
        e.target.style.opacity = "1";
    }
</script>

<div class="scrollable">
    <div class="grid" class:dimmed={isProcessing}>
        {#each editorStore.images as img, index (img.id)}
            <ImageCard
                {img}
                {isProcessing}
                onRemove={() => editorStore.removeImageByIndex(index)}
                onDragStart={(e) => handleDragStart(e, index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDragEnd={handleDragEnd}
            />
        {/each}

        {#if editorStore.images.length === 0}
            <div class="empty-state">
                <p>Выберите файлы или папку</p>
            </div>
        {/if}
    </div>
</div>

<style>
    .scrollable {
        flex: 1;
        overflow-y: auto;
        width: 100%;
    }
    .grid {
        display: grid;
        /* Колонки ровно 150px, сколько влезет в ряд */
        grid-template-columns: repeat(auto-fill, 150px);

        /* ВАЖНО: Высота ряда подстраивается под содержимое */
        grid-auto-rows: auto;

        gap: 15px;
        padding: 1.5rem;

        /* ВАЖНО: Прижимаем карточки к верху, чтобы они не растягивались на всю высоту ряда */
        align-items: start;

        /* Центрируем всю сетку, если сбоку остается место */
        justify-content: center;

        flex: 1;
    }
    .grid.dimmed {
        opacity: 0.5;
        pointer-events: none;
    }

    .empty-state {
        grid-column: 1 / -1;
        display: flex;
        justify-content: center;
        align-items: center;
        height: 300px;
        color: #555;
        border: 2px dashed #333;
        border-radius: 10px;
    }
</style>
