<script>
    import ImageCard from "./ImageCard.svelte";

    import { images, removeImageByIndex } from "../stores/appStore.js";

    export let isProcessing;

    let draggedIndex = null;
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

        // Логика перестановки прямо в сторе
        images.update((currentImages) => {
            const newImages = [...currentImages];
            const item = newImages[sourceIdx];
            newImages.splice(sourceIdx, 1);
            newImages.splice(targetIdx, 0, item);
            return newImages;
        });

        draggedIndex = targetIdx;
    }

    function handleDragEnd(e) {
        draggedIndex = null;
        e.target.style.opacity = "1";
    }
</script>

<div class="grid" class:dimmed={isProcessing}>
    {#each $images as img, index (img.id)}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <ImageCard
            {img}
            {index}
            {isProcessing}
            on:removeImage={() => removeImageByIndex(index)}
            on:dragstart={(e) => handleDragStart(e, index)}
            on:dragover={(e) => handleDragOver(e, index)}
            on:dragend={handleDragEnd}
        />
    {/each}

    {#if $images.length === 0}
        <div class="empty-state">
            <p>Перетащите сюда файлы или выберите папку</p>
        </div>
    {/if}
</div>

<style>
    .grid {
        display: grid;
        /* Минимальная ширина 150px, карточки будут заполнять пространство */
        grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
        gap: 15px;
        padding: 1.5rem;
        overflow-y: auto;
        flex: 1;
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
