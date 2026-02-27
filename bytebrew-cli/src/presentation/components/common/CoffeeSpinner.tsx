// Coffee-themed spinner with multiple beautiful variants
import React, { useState, useEffect } from 'react';
import { Text } from 'ink';
import cliSpinners from 'cli-spinners';
import { colors } from '../../theme/colors.js';

const COFFEE_PHRASES = [
  'Brewing',
  'Synthesizing',
];

// Custom coffee cup with animated steam - compact version
const COFFEE_CUP_FRAMES = [
  '  ∘  \n ☕ ',
  '  ○  \n ☕ ',
  ' ∘   \n ☕ ',
  '   ° \n ☕ ',
  '  °  \n ☕ ',
  ' °   \n ☕ ',
  '   ∘ \n ☕ ',
  '  ∙  \n ☕ ',
];

// Simple inline coffee steam
const COFFEE_STEAM_FRAMES = [
  '☕～',
  '☕≈',
  '☕∿',
  '☕≋',
  '☕～',
  '☕∾',
];

// Latte art style - filling/pouring effect
const LATTE_ART_FRAMES = [
  '○',
  '◔',
  '◑',
  '◕',
  '●',
  '◉',
  '◎',
  '○',
];

// Bean roasting animation
const BEAN_FRAMES = [
  '⚬',
  '⚭',
  '⚮',
  '⚯',
  '●',
  '⦿',
  '◉',
  '◎',
];

// Aesthetic dots - short 3-element version
const AESTHETIC_FRAMES = [
  '▰▱▱',
  '▰▰▱',
  '▰▰▰',
  '▱▰▰',
  '▱▱▰',
  '▱▱▱',
];

// Moon phases - like coffee getting ready
const MOON_FRAMES = cliSpinners.moon.frames;

// Material design loading
const MATERIAL_FRAMES = cliSpinners.material.frames;

// Bouncing ball - energetic like caffeine
const BOUNCE_FRAMES = cliSpinners.bouncingBall.frames;

type SpinnerVariant =
  | 'steam'      // ☕～ with steam waves
  | 'cup'        // Coffee cup with rising steam
  | 'latte'      // Filling circle (latte art)
  | 'bean'       // Coffee bean animation
  | 'aesthetic'  // Aesthetic braille dots
  | 'moon'       // Moon phases
  | 'material'   // Material design
  | 'bounce';    // Bouncing ball (caffeine energy)

const VARIANT_FRAMES: Record<SpinnerVariant, string[]> = {
  steam: COFFEE_STEAM_FRAMES,
  cup: COFFEE_CUP_FRAMES,
  latte: LATTE_ART_FRAMES,
  bean: BEAN_FRAMES,
  aesthetic: AESTHETIC_FRAMES,
  moon: MOON_FRAMES,
  material: MATERIAL_FRAMES,
  bounce: BOUNCE_FRAMES,
};

const VARIANT_INTERVALS: Record<SpinnerVariant, number> = {
  steam: 200,
  cup: 250,
  latte: 150,
  bean: 180,
  aesthetic: 80,
  moon: 100,
  material: 17,
  bounce: 80,
};

interface CoffeeSpinnerProps {
  variant?: SpinnerVariant;
  phraseInterval?: number;
  showPhrase?: boolean;
  color?: string;
  actionLabel?: string;
}

export const CoffeeSpinner: React.FC<CoffeeSpinnerProps> = ({
  variant = 'steam',
  phraseInterval = 4000,
  showPhrase = true,
  color = colors.processing,
  actionLabel,
}) => {
  const frames = VARIANT_FRAMES[variant];
  const interval = VARIANT_INTERVALS[variant];

  const [frameIndex, setFrameIndex] = useState(0);
  const [phraseIndex, setPhraseIndex] = useState(
    Math.floor(Math.random() * COFFEE_PHRASES.length)
  );

  useEffect(() => {
    const timer = setInterval(() => {
      setFrameIndex((prev) => (prev + 1) % frames.length);
    }, interval);

    return () => clearInterval(timer);
  }, [interval, frames.length]);

  useEffect(() => {
    if (!showPhrase) return;

    const timer = setInterval(() => {
      setPhraseIndex((prev) => (prev + 1) % COFFEE_PHRASES.length);
    }, phraseInterval);

    return () => clearInterval(timer);
  }, [phraseInterval, showPhrase]);

  const frame = frames[frameIndex];
  const phrase = COFFEE_PHRASES[phraseIndex];

  // For multiline frames (cup variant), render differently
  if (variant === 'cup') {
    const lines = frame.split('\n');
    return (
      <Text color={color}>
        {lines[0]}
        {showPhrase && ` ${actionLabel || phrase}...`}
        {'\n'}
        {lines[1]}
      </Text>
    );
  }

  return (
    <Text color={color}>
      {frame} {showPhrase && `${actionLabel || phrase}...`}
    </Text>
  );
};

// Export variants for easy access
export const SPINNER_VARIANTS: SpinnerVariant[] = [
  'steam',
  'latte',
  'bean',
  'aesthetic',
  'moon',
  'material',
  'bounce',
];
