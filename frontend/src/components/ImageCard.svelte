<script>
    let { img, isProcessing, onRemove, onDragStart, onDragOver, onDragEnd } =
        $props();

    function handleRemoveClick(e) {
        e.stopPropagation();
        onRemove?.()
    }
</script>

<div
    class="card"
    class:selected={img.selected}
    draggable={!isProcessing}
    role="listitem"
    ondragstart={onDragStart}
    ondragover={onDragOver}
    ondragend={onDragEnd}
>
    <div class="card-inner">
        <button
            class="close-btn"
            onclick={handleRemoveClick}
            title="Убрать из списка">×</button
        >

        <div class="checkbox-wrapper">
            <input type="checkbox" bind:checked={img.selected} />
        </div>

        <div class="img-wrapper">
            <img src={img.thumbnailSrc} alt={img.name} />
        </div>

        <div class="name">{img.name}</div>
    </div>
</div>

<style>
    .card {
        background: var(--card-bg);
        border-radius: 6px;
        width: 100%;
        height: auto;
        position: relative;
        z-index: 0;
        cursor: grab;
        border: 2px solid transparent;
        transition: border-color 0.1s;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .card:active {
        cursor: grabbing;
    }
    .card:not(.selected) {
        opacity: 0.5;
        filter: grayscale(1);
    }
    .card.selected:hover {
        border-color: var(--accent);
    }

    .card-inner {
        width: 100%;
        height: auto;
        display: flex;
        flex-direction: column;
    }

    .img-wrapper {
        width: 100%;
        height: auto;
        display: block;
        line-height: 0;
    }

    img {
        width: 100%;
        height: auto;
        display: block;
        object-fit: contain;
    }

    .name {
        padding: 6px;
        font-size: 0.75rem;
        color: #888;
        background: #252525;
        text-align: center;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        width: 100%;
    }

    .close-btn {
        position: absolute;
        top: 6px;
        right: 6px;
        z-index: 10;
        background: rgba(0, 0, 0, 0.6);
        color: #fff;
        border: none;
        width: 24px;
        height: 24px;
        border-radius: 50%;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.2s;
        font-size: 16px;
    }

    .card:hover .close-btn {
        opacity: 1;
    }
    .close-btn:hover {
        background: #ff4444;
    }

    .checkbox-wrapper {
        position: absolute;
        top: 6px;
        left: 6px;
        z-index: 10;
    }

    .checkbox-wrapper input {
        width: 18px;
        height: 18px;
        cursor: pointer;
        accent-color: var(--accent);
    }
</style>
