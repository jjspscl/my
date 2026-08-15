// Minimal ambient types for the optional local sample-validation test.
// The browser tsconfig does not include @types/node; only the two fs
// functions used by gcash-sample.test.ts are declared.
declare module 'node:fs' {
  export function readFileSync(path: string): Uint8Array
  export function readdirSync(path: string): string[]
}
