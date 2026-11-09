// What each kind of event changes in the app. The keys match the API's
// occasions.

export type Tone = { key: string; label: string };

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
  // Celebrations get confetti and the playful tone.
  celebration: boolean;
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
