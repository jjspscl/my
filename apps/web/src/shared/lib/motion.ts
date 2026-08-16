import { useReducedMotion } from 'motion/react'
import type { TargetAndTransition, Variants } from 'motion/react'

// Shared entrance variants following the house style across finance pages:
// opacity + small y offset, no explicit transition (motion defaults).
// Containers stagger their children; items and form fields fade up.
const CONTAINER: Variants = {
  animate: { transition: { staggerChildren: 0.04 } },
}
const ITEM: { initial: TargetAndTransition; animate: TargetAndTransition } = {
  initial: { opacity: 0, y: 4 },
  animate: { opacity: 1, y: 0 },
}
const FIELD: { initial: TargetAndTransition; animate: TargetAndTransition } = {
  initial: { opacity: 0, y: 6 },
  animate: { opacity: 1, y: 0 },
}

// Zeroed variants: rendered unchanged when the user prefers reduced motion.
const ZEROED: { initial: TargetAndTransition; animate: TargetAndTransition } = {
  initial: { opacity: 1, y: 0 },
  animate: { opacity: 1, y: 0 },
}

export interface MotionPreset {
  container: Variants
  item: { initial: TargetAndTransition; animate: TargetAndTransition }
  field: { initial: TargetAndTransition; animate: TargetAndTransition }
}

// useMotionPreset returns entrance variants gated on prefers-reduced-motion:
// users who ask for reduced motion get static content, everyone else gets the
// staggered fade-up. jsdom has no matchMedia; motion's useReducedMotion
// guards that internally and reports false, so tests are unaffected.
export function useMotionPreset(): MotionPreset {
  const reduced = useReducedMotion()
  if (reduced) {
    return { container: ZEROED, item: ZEROED, field: ZEROED }
  }
  return { container: CONTAINER, item: ITEM, field: FIELD }
}
