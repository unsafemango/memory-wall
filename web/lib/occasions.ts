// What each kind of event changes in the app. The keys match the API's
// occasions.

export type Tone = { key: string; label: string };

export type ConfettiShape = "strip" | "round" | "heart" | "star" | "petal";

// How an occasion's confetti looks. speed 1 is a normal fall, lower is gentler.
export type ConfettiEffect = {
  colors: string[];
  shapes: ConfettiShape[];
  count: number;
  speed: number;
};

export const DEFAULT_EFFECT: ConfettiEffect = {
  colors: ["#ff5da2", "#8b5cf6", "#ffb347", "#ffd93d", "#5ec8f0"],
  shapes: ["strip", "strip", "round"],
  count: 36,
  speed: 1,
};

export type Occasion = {
  key: string;
  label: string;
  // Example event name for the create form.
  titleExample: string;
  // Prompt for who the event is for, and an example answer.
  honoreeLabel: string;
  honoreeExample: string;
  // Shown under the title on the event page.
  heroLine: string;
  // Shown in the message box on the guest page.
  messagePrompt: string;
  signoff: string;
  // Celebrations get the playful tone, a countdown, and confetti.
  celebration: boolean;
  // Left out for occasions that shouldn't have any.
  effect?: ConfettiEffect;
  tones: Tone[];
};

const celebrationTones: Tone[] = [
  { key: "heartfelt", label: "Heartfelt" },
  { key: "funny", label: "Funny" },
  { key: "short", label: "Short and sweet" },
];

const celebration = { celebration: true, tones: celebrationTones, signoff: "With love, always" };

export const OCCASIONS: Occasion[] = [
  {
    ...celebration,
    key: "birthday",
    effect: DEFAULT_EFFECT,
    label: "Birthday",
    titleExample: "Sarah's 30th Birthday",
    honoreeLabel: "Whose birthday is it?",
    honoreeExample: "Sarah",
    heroLine: "A few people who love you wanted to leave you something.",
    messagePrompt: "Share a birthday wish, a memory, or what they mean to you",
  },
  {
    ...celebration,
    key: "wedding",
    effect: { colors: ["#f7b7c5", "#e8799b", "#e3c179", "#d9a0c9"], shapes: ["petal", "petal", "heart"], count: 26, speed: 0.6 },
    label: "Wedding",
    titleExample: "Alex & Jordan's Wedding",
    honoreeLabel: "Who's getting married?",
    honoreeExample: "Alex & Jordan",
    heroLine: "Love, memories, and good wishes from the people around you.",
    messagePrompt: "Share your wishes for the couple, or a favourite memory of them",
  },
  {
    ...celebration,
    key: "anniversary",
    effect: { colors: ["#e3b04b", "#e8799b", "#c9963a", "#f2c4ce"], shapes: ["heart", "round", "strip"], count: 24, speed: 0.65 },
    label: "Anniversary",
    titleExample: "Mom & Dad's 40th Anniversary",
    honoreeLabel: "Whose anniversary is it?",
    honoreeExample: "Mom & Dad",
    heroLine: "Memories and love from the people who've watched your story.",
    messagePrompt: "Share a memory of them together, or your wishes for the years ahead",
  },
  {
    ...celebration,
    key: "graduation",
    effect: { colors: ["#f2b632", "#6d28d9", "#2c3a8c", "#e8a01c"], shapes: ["star", "star", "strip"], count: 34, speed: 0.95 },
    label: "Graduation",
    titleExample: "Maya's Graduation",
    honoreeLabel: "Who's graduating?",
    honoreeExample: "Maya",
    heroLine: "A few people who are proud of you wanted to say so.",
    messagePrompt: "Share why you're proud of them, or your wishes for what's next",
  },
  {
    ...celebration,
    key: "baby",
    effect: { colors: ["#a8dcf2", "#f7c6d9", "#fbe39a", "#c5e8c9", "#d6c8f5"], shapes: ["star", "round", "round"], count: 24, speed: 0.55 },
    label: "New baby",
    titleExample: "Welcome, Baby Rivera",
    honoreeLabel: "Who is it for?",
    honoreeExample: "The Rivera family",
    heroLine: "Love and good wishes for your growing family.",
    messagePrompt: "Share your wishes for the baby and the family",
  },
  {
    ...celebration,
    key: "retirement",
    effect: { colors: ["#e3b04b", "#f2c14e", "#d9823b", "#7c9a6d"], shapes: ["strip", "star", "round"], count: 26, speed: 0.75 },
    label: "Retirement",
    titleExample: "Dana's Retirement",
    honoreeLabel: "Who's retiring?",
    honoreeExample: "Dana",
    heroLine: "Thank-yous and memories from the people you've worked with.",
    messagePrompt: "Share a memory, a thank-you, or your wishes for what's next",
  },
  {
    ...celebration,
    key: "farewell",
    effect: { colors: ["#5ec8f0", "#38b2ac", "#ffb347", "#8b5cf6"], shapes: ["strip", "round"], count: 28, speed: 0.8 },
    label: "Farewell",
    titleExample: "Goodbye and Good Luck, Sam",
    honoreeLabel: "Who's moving on?",
    honoreeExample: "Sam",
    heroLine: "Memories and good wishes to take with you.",
    messagePrompt: "Share a memory, a thank-you, or your wishes for what's next",
  },
  {
    key: "memorial",
    label: "Memorial",
    titleExample: "Remembering Rose",
    honoreeLabel: "Who are you remembering?",
    honoreeExample: "Rose",
    heroLine: "Memories from the people who loved them.",
    messagePrompt: "Share a memory, or what they meant to you",
    signoff: "Always remembered",
    celebration: false,
    tones: [
      { key: "heartfelt", label: "Heartfelt" },
      { key: "memory", label: "A favourite memory" },
      { key: "gentle", label: "Short and gentle" },
    ],
  },
  {
    ...celebration,
    key: "other",
    effect: DEFAULT_EFFECT,
    label: "Something else",
    titleExample: "Our Team's Big Win",
    honoreeLabel: "Who is it for?",
    honoreeExample: "The whole team",
    heroLine: "A few people wanted to leave you something.",
    messagePrompt: "Share a memory, a wish, or what they mean to you",
  },
];

const fallback = OCCASIONS[OCCASIONS.length - 1];

export function occasionFor(key?: string): Occasion {
  return OCCASIONS.find((o) => o.key === key) ?? fallback;
}
