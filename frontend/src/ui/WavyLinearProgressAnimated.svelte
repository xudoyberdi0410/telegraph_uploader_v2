<script lang="ts">
  import { Tween } from "svelte/motion";
  import { cubicOut, cubicInOut } from "svelte/easing";
  import { linear, trackOpacity } from "../utils/wavy";

  interface Props {
    width?: number;
    height?: number;
    thickness?: number;
    percent: number;
  }

  let {
    width = 600,
    height = 10,
    thickness = 4,
    percent = 0,
  }: Props = $props();

  // 1. Создаем экземпляр Tween (это класс в Svelte 5)
  const smoothPercent = new Tween(percent, {
    duration: 1000,
    easing: cubicInOut,
  });

  // 2. Синхронизируем Tween при изменении пропса.
  // Вместо .set() теперь просто обновляем свойство .target
  $effect(() => {
    smoothPercent.target = percent;
  });

  let left = $derived(thickness * 0.5);
  let right = $derived(width - thickness * 0.5);

  // 3. Используем .current для получения текущего анимированного значения
  let currentX = $derived((smoothPercent.current / 100) * (right - left) + left);

  let d = $state("");

  $effect(() => {
    let frame: number;

    const update = () => {
      const time = performance.now();

      // currentX автоматически отслеживает изменения smoothPercent.current
      d = linear(
        height / 2 - thickness / 2,
        height / 2,
        left,
        currentX,
        time
      );

      frame = requestAnimationFrame(update);
    };

    frame = requestAnimationFrame(update);
    return () => cancelAnimationFrame(frame);
  });
</script>

<svg viewBox="0 0 {width} {height}">
  <path
    {d}
    fill="none"
    stroke="var(--m3c-primary)"
    stroke-width={thickness}
    stroke-linecap="round"
  />
  
  <line
    fill="none"
    stroke="var(--m3c-secondary-container)"
    stroke-width={thickness}
    stroke-linecap="round"
    x1={currentX + thickness + 4}
    y1={height / 2}
    x2={right}
    y2={height / 2}
    opacity={trackOpacity(right, currentX + thickness + 4)}
  />
</svg>
