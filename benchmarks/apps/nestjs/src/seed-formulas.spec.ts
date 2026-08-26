import {
  projectDescription,
  projectName,
  projectOwnerId,
  userEmail,
  userName,
} from './seed-formulas';

// Pure, no-database checks of the seed content formulas — port of
// benchmarks/apps/shared/seed_test.go (also ported to the Django/Rails/Laravel
// suites). This app can't import any of them, so the same properties are
// re-verified against this app's own from-scratch port.
describe('seed formulas', () => {
  it('are deterministic', () => {
    expect(userEmail(1)).toBe('user-0001@example.com');
    expect(userName(1)).toBe('User 0001');
    expect(userEmail(1000)).toBe('user-1000@example.com');
    expect(projectName(1)).toBe('Project 000001');
    expect(projectDescription(1)).toBe('Seeded benchmark project 000001');
    expect(projectName(100000)).toBe('Project 100000');
  });

  it('round-robin owner id wraps correctly', () => {
    const userCount = 7;
    expect(projectOwnerId(1, userCount)).toBe(1);
    expect(projectOwnerId(7, userCount)).toBe(7);
    // The 8th project wraps back to owner 1 — the round-robin boundary an
    // off-by-one would break silently.
    expect(projectOwnerId(8, userCount)).toBe(1);
    expect(projectOwnerId(14, userCount)).toBe(7);
    expect(projectOwnerId(15, userCount)).toBe(1);
  });
});
