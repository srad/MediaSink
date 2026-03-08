/**
 * This is a quadratic function which starts slowly and slows down slowly.
 * @param t
 */
export function easeInOutQuad(t: number): number {
  return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
}

/**
 * Scroll smoothly to the specified left pixel distance for any given element.
 * Notice that the element must also be scrollable (overflow).
 * @param element Element reference.
 * @param left Pixel to scroll to left.
 * @param duration In which timespan shall the animation be completed.
 */
export function animateScrollLeft(element: Element, left: number, duration: number) {
  const start = element.scrollLeft;
  const startTime = performance.now();
  const distance = left - start;

  function scrollStep(currentTime: number) {
    const elapsed = currentTime - startTime;
    const progress = Math.min(elapsed / duration, 1); // Clamp progress between 0 and 1
    const easedProgress = easeInOutQuad(progress); // Apply easing

    element.scrollLeft = start + distance * easedProgress;

    if (progress < 1) {
      requestAnimationFrame(scrollStep);
    }
  }

  requestAnimationFrame(scrollStep);
}
