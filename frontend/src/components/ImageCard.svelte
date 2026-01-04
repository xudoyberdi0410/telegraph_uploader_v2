<script>
    export let img;
    export let isProcessing;

    import { createEventDispatcher } from "svelte";
    const dispatch = createEventDispatcher();
</script>

<div
    class="card"
    class:selected={img.selected}
    draggable={!isProcessing}
    on:dragstart
    on:dragover
    on:dragend
>
    <div class="card-inner">
        <!-- Крестик удаления -->
        <button
            class="close-btn"
            on:click|stopPropagation={() => dispatch("removeImage")}
            title="Убрать из списка">×</button
        >

        <!-- Чекбокс -->
        <div class="checkbox-wrapper">
            <input type="checkbox" bind:checked={img.selected} />
        </div>

        <!-- Картинка -->
        <div class="img-wrapper">
            <img
                src={img.thumbnailSrc}
                alt={img.name}
                loading="lazy"
                decoding="async"
            />
        </div>

        <div class="name">{img.name}</div>
    </div>
</div>

<style>
    .card {
        background: var(--card-bg);
        border-radius: 6px;
        
        /* Ширина фиксируется сеткой, высота — авто */
        width: 100%;
        height: auto; 
        
        position: relative;
        z-index: 0;
        cursor: grab;
        border: 2px solid transparent;
        transition: border-color 0.1s;
        
        /* Чтобы скругления углов работали */
        overflow: hidden; 
        
        /* Убираем лишние отступы у блочных элементов */
        display: flex;
        flex-direction: column;
    }

    /* Убираем aspect-ratio, который мог вызывать наложение! */

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
        /* Высота должна быть авто, не 100% */
        height: auto;
        display: flex;
        flex-direction: column;
    }

    .img-wrapper {
        width: 100%;
        height: auto;
        display: block; /* Убираем лишние флекс-отступы */
        line-height: 0; /* Убираем отступ под картинкой */
    }

    img {
        width: 100%;
        height: auto; /* Картинка определяет высоту */
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
        
        /* Имя всегда снизу */
        width: 100%;
    }

    /* Кнопки и чекбоксы остаются без изменений */
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
