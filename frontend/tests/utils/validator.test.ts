import { describe, it, expect } from "vitest";
import { FieldConfig, createValidator } from "../../src/utils/validator";

describe("FieldValidator", () => {
  const fieldConfigs: FieldConfig[] = [
    {
      name: "username",
      validator: (t) => typeof t === "string" && t.length >= 3,
      validMessage: "Valid username",
      invalidMessage: "Invalid username, must be at least 3 characters",
    },
    {
      name: "age",
      validator: (t) => typeof t === "number" && t > 0,
      validMessage: "Valid age",
      invalidMessage: "Invalid age, must be a positive number",
    },
    {
      name: "isActive",
      validator: (t) => typeof t === "boolean",
      validMessage: "Valid status",
      invalidMessage: "Invalid status, must be boolean",
    },
  ];

  const validator = createValidator(fieldConfigs);

  it("should validate a correct username", () => {
    const result = validator.validate("username", "john");
    expect(result).toEqual({ isValid: true, message: 'Valid username: "john"' });
  });

  it("should invalidate a short username", () => {
    const result = validator.validate("username", "ab");
    expect(result).toEqual({ isValid: false, message: 'Invalid username, must be at least 3 characters: "ab"' });
  });

  it("should validate a positive age", () => {
    const result = validator.validate("age", 25);
    expect(result).toEqual({ isValid: true, message: 'Valid age: "25"' });
  });

  it("should invalidate a negative age", () => {
    const result = validator.validate("age", -5);
    expect(result).toEqual({ isValid: false, message: 'Invalid age, must be a positive number: "-5"' });
  });

  it("should invalidate a non-numeric age", () => {
    const result = validator.validate("age", "twenty");
    expect(result).toEqual({ isValid: false, message: 'Invalid age, must be a positive number: "twenty"' });
  });

  it("should validate a boolean isActive", () => {
    const result = validator.validate("isActive", true);
    expect(result).toEqual({ isValid: true, message: 'Valid status: "true"' });
  });

  it("should invalidate a non-boolean isActive", () => {
    const result = validator.validate("isActive", "yes");
    expect(result).toEqual({ isValid: false, message: 'Invalid status, must be boolean: "yes"' });
  });

  it("should throw an error for an unknown field", () => {
    expect(() => validator.validate("unknownField", "test")).toThrowError("Invalid field config: undefined");
  });

  it("should validate multiple fields correctly", () => {
    const results = validator.validateAll([
      { name: "username", value: "johndoe" },
      { name: "age", value: 30 },
      { name: "isActive", value: false },
    ]);

    expect(results).toEqual([
      { isValid: true, message: 'Valid username: "johndoe"' },
      { isValid: true, message: 'Valid age: "30"' },
      { isValid: true, message: 'Valid status: "false"' },
    ]);
  });

  it("should handle mixed valid and invalid values in validateAll", () => {
    const results = validator.validateAll([
      { name: "username", value: "jo" },
      { name: "age", value: -1 },
      { name: "isActive", value: "maybe" },
    ]);

    expect(results).toEqual([
      { isValid: false, message: 'Invalid username, must be at least 3 characters: "jo"' },
      { isValid: false, message: 'Invalid age, must be a positive number: "-1"' },
      { isValid: false, message: 'Invalid status, must be boolean: "maybe"' },
    ]);
  });
});
