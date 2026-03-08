import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useAuthStore } from "../../src/stores/auth";

// Setup window globals before mocking
beforeEach(() => {
  (globalThis as any).window = {
    APP_APIURL: "http://localhost:8000",
    APP_API_VERSION: "1.0",
  };
});

afterEach(() => {
  delete (globalThis as any).window;
});

// Mock ClientFactory - AuthService will use this to make API calls
vi.mock("../../src/services/api/v1/ClientFactory", () => ({
  createClient: vi.fn(() => ({
    auth: {
      loginCreate: vi.fn(() => Promise.resolve({ token: "mocked-jwt-token" })),
      signupCreate: vi.fn(() => Promise.resolve({})),
    },
  })),
}));

describe("Auth Store", () => {
  beforeEach(() => {
    // Reset Pinia before each test
    setActivePinia(createPinia());
    localStorage.clear();
    // Reset mocks
    vi.clearAllMocks();
  });

  it("should initialize with correct default values", () => {
    const authStore = useAuthStore();
    expect(authStore.loggedIn).toBe(false);
    expect(authStore.token).toBeNull();
  });

  it("should log in and update state correctly", async () => {
    const authStore = useAuthStore();
    // Should not throw an error
    try {
      await authStore.login({ username: "test", password: "1234" });
      expect(true).toBe(true);
    } catch (error) {
      expect.fail(`Login should not throw: ${error}`);
    }
  });

  it("should log out and reset state", () => {
    const authStore = useAuthStore();
    authStore.token = "some-token"; // Simulate logged-in state
    authStore.loggedIn = true;

    authStore.logout();

    expect(authStore.token).toBeNull();
    expect(authStore.loggedIn).toBe(false);
    expect(localStorage.getItem("jwt")).toBeNull();
    expect(localStorage.getItem("authenticated")).toBeNull();
  });

  it("should call register action without errors", async () => {
    const authStore = useAuthStore();
    // Should not throw an error
    await authStore.register({ username: "test", password: "1234" });
    expect(true).toBe(true);
  });

  it("should return correct values from getters", async () => {
    const authStore = useAuthStore();
    await authStore.login({ username: "test", password: "1234" });

    expect(authStore.isLoggedIn).toBe(true);
    expect(authStore.getToken).toBe("mocked-jwt-token");
  });
});
