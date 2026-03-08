import { describe, expect, it } from "vitest";
import { parseTagsString, validateTags } from "../../src/utils/parser";

describe("validateTags", function () {
  it("valid tags return true", function () {
    expect(validateTags([])).toBeTruthy();
    expect(validateTags(["test"])).toBeTruthy();
    expect(validateTags(["test", "test2"])).toBeTruthy();
    expect(validateTags(["test-0", "test", "test-2"])).toBeTruthy();
  });

  it("invalid tags return false when whites space wrong", function () {
    expect(validateTags([""])).toBeFalsy();
    expect(validateTags([" "])).toBeFalsy();
    expect(validateTags(["  "])).toBeFalsy();
    expect(validateTags(["", ""])).toBeFalsy();
    expect(validateTags([" test"])).toBeFalsy();
    expect(validateTags(["t est"])).toBeFalsy();
    expect(validateTags([" test "])).toBeFalsy();
  });

  it("invalid tags when dashes placed wrong", function () {
    expect(validateTags(["test-"])).toBeFalsy();
    expect(validateTags(["-test"])).toBeFalsy();
    expect(validateTags(["-test-"])).toBeFalsy();
    expect(validateTags(["-te-st-"])).toBeFalsy();

    expect(validateTags(["test-", "-test"])).toBeFalsy();
    expect(validateTags(["-test", "-test-"])).toBeFalsy();
    expect(validateTags(["-test-", "-te-st-"])).toBeFalsy();
    expect(validateTags(["-te-st-", "-te-st-"])).toBeFalsy();
  });

  it("is valid when dashes placed right", function () {
    expect(validateTags(["t-est"])).toBeTruthy();
    expect(validateTags(["tes-t"])).toBeTruthy();
    expect(validateTags(["t-est"])).toBeTruthy();
    expect(validateTags(["t-e-s-t"])).toBeTruthy();

    expect(validateTags(["t-est", "tes-t"])).toBeTruthy();
    expect(validateTags(["tes-t", "t-est"])).toBeTruthy();
    expect(validateTags(["t-est", "t-e-s-t"])).toBeTruthy();
    expect(validateTags(["t-e-s-t", "t-e-s-t"])).toBeTruthy();
  });
});

describe("parseTagsString", function () {
  it("empty string generates empty array", function () {
    expect(parseTagsString("")).to.deep.equal([]);
  });

  it("only white spaces raises an error", function () {
    const parse = () => parseTagsString("          ");
    expect(parse).toThrow();
  });

  it("single valid tag", function () {
    expect(parseTagsString("a")).to.deep.equal(["a"]);
  });

  it("single valid tag", function () {
    expect(parseTagsString("aa")).to.deep.equal(["aa"]);
  });

  it("single valid tag", function () {
    expect(parseTagsString("aabb")).to.deep.equal(["aabb"]);
  });

  it("two valid tags", function () {
    expect(parseTagsString("a,a")).to.deep.equal(["a", "a"]);
  });

  it("two valid tags", function () {
    expect(parseTagsString("aa,bb")).to.deep.equal(["aa", "bb"]);
  });

  it("single invalid tag", function () {
    const parse = () => parseTagsString("l#üp");

    expect(parse).toThrow();
  });

  it("two invalid tags raise error", function () {
    const parse = () => parseTagsString("l#üp,389äö");
    expect(parse).toThrow();
  });
});
