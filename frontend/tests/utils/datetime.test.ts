import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";
import { fromNow } from "../../src/utils/datetime";

describe("fromNow (using moment.js)", () => {
  let now: number;

  beforeEach(() => {
    now = Date.now();
    vi.useFakeTimers();
    vi.setSystemTime(now);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("returns human-readable relative time", () => {
    it("should return 'a few seconds ago' for very recent times", () => {
      const result = fromNow(now - 1000);
      expect(result).toMatch(/ago/i);
    });

    it("should return 'a minute ago' for ~1 minute ago", () => {
      const result = fromNow(now - 60 * 1000);
      expect(result).toMatch(/minute/i);
    });

    it("should return 'minutes ago' for ~5 minutes ago", () => {
      const result = fromNow(now - 5 * 60 * 1000);
      expect(result).toMatch(/(minute|min)/i);
    });

    it("should return 'an hour ago' for ~1 hour ago", () => {
      const result = fromNow(now - 60 * 60 * 1000);
      expect(result).toMatch(/hour/i);
    });

    it("should return 'hours ago' for ~5 hours ago", () => {
      const result = fromNow(now - 5 * 60 * 60 * 1000);
      expect(result).toMatch(/hour/i);
    });

    it("should return 'a day ago' for ~1 day ago", () => {
      const result = fromNow(now - 24 * 60 * 60 * 1000);
      expect(result).toMatch(/day/i);
    });

    it("should return 'days ago' for ~5 days ago", () => {
      const result = fromNow(now - 5 * 24 * 60 * 60 * 1000);
      expect(result).toMatch(/day/i);
    });

    it("should return 'a month ago' for ~30 days ago", () => {
      const result = fromNow(now - 30 * 24 * 60 * 60 * 1000);
      expect(result).toMatch(/month/i);
    });

    it("should return 'months ago' for ~90 days ago", () => {
      const result = fromNow(now - 90 * 24 * 60 * 60 * 1000);
      expect(result).toMatch(/month/i);
    });

    it("should return 'a year ago' for ~365 days ago", () => {
      const result = fromNow(now - 365 * 24 * 60 * 60 * 1000);
      expect(result).toMatch(/year/i);
    });

    it("should return 'years ago' for ~730 days ago", () => {
      const result = fromNow(now - 730 * 24 * 60 * 60 * 1000);
      expect(result).toMatch(/year/i);
    });
  });

  describe("returns string with 'ago'", () => {
    it("should always include 'ago' in the result", () => {
      expect(fromNow(now - 1000)).toMatch(/ago/i);
      expect(fromNow(now - 60 * 1000)).toMatch(/ago/i);
      expect(fromNow(now - 24 * 60 * 60 * 1000)).toMatch(/ago/i);
      expect(fromNow(now - 365 * 24 * 60 * 60 * 1000)).toMatch(/ago/i);
    });

    it("should return a string", () => {
      expect(typeof fromNow(now)).toBe("string");
      expect(typeof fromNow(now - 1000)).toBe("string");
      expect(typeof fromNow(now - 60 * 1000)).toBe("string");
    });
  });

  describe("edge cases", () => {
    it("should handle timestamps in milliseconds", () => {
      const oneMinuteAgo = now - 60000;
      const result = fromNow(oneMinuteAgo);
      expect(result).toMatch(/minute/i);
    });

    it("should handle Date.parse results", () => {
      const dateString = new Date(now - 24 * 60 * 60 * 1000).toISOString();
      const parsedTime = Date.parse(dateString);
      const result = fromNow(parsedTime);
      expect(result).toMatch(/day/i);
    });

    it("should handle very recent times (current time)", () => {
      const result = fromNow(now);
      expect(result).toBeDefined();
      expect(typeof result).toBe("string");
      expect(result.length).toBeGreaterThan(0);
    });

    it("should return consistent results for the same input", () => {
      const testTime = now - 5 * 60 * 1000;
      const result1 = fromNow(testTime);
      const result2 = fromNow(testTime);
      expect(result1).toBe(result2);
    });
  });

  describe("moment.js specific behavior", () => {
    it("should use moment.js for formatting", () => {
      // Test that it uses moment.js conventions
      // moment.js rounds values, so 90 seconds might show as "2 minutes"
      const result = fromNow(now - 90 * 1000);
      expect(result).toMatch(/minute/i);
    });

    it("should handle year boundaries correctly", () => {
      const almostAYear = now - 300 * 24 * 60 * 60 * 1000;
      const result = fromNow(almostAYear);
      expect(result).toMatch(/(month|year)/i);
    });

    it("should handle month boundaries correctly", () => {
      const almostAMonth = now - 25 * 24 * 60 * 60 * 1000;
      const result = fromNow(almostAMonth);
      expect(result).toMatch(/(day|month)/i);
    });
  });
});
