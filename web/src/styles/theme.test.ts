import { describe, it, expect } from 'vitest';
import { colors, shadows, cardStyles, dialogStyles, buttonColors, badgeColors } from './theme';

describe('theme', () => {
  describe('colors', () => {
    it('exports background colors', () => {
      expect(colors.bgDefault).toBe('var(--bgColor-default)');
      expect(colors.bgMuted).toBe('var(--bgColor-muted)');
      expect(colors.bgInset).toBe('var(--bgColor-inset)');
    });

    it('exports foreground colors', () => {
      expect(colors.fgDefault).toBe('var(--fgColor-default)');
      expect(colors.fgMuted).toBe('var(--fgColor-muted)');
      expect(colors.fgAccent).toBe('var(--fgColor-accent)');
    });

    it('exports status colors', () => {
      expect(colors.fgSuccess).toBe('var(--fgColor-success)');
      expect(colors.fgDanger).toBe('var(--fgColor-danger)');
      expect(colors.fgWarning).toBe('var(--fgColor-attention)');
    });

    it('exports border colors', () => {
      expect(colors.borderDefault).toBe('var(--borderColor-default)');
      expect(colors.borderMuted).toBe('var(--borderColor-muted)');
    });
  });

  describe('shadows', () => {
    it('exports shadow values', () => {
      expect(shadows.small).toBe('var(--shadow-resting-small)');
      expect(shadows.medium).toBe('var(--shadow-resting-medium)');
      expect(shadows.large).toBe('var(--shadow-floating-large)');
    });
  });

  describe('cardStyles', () => {
    it('exports base card styles', () => {
      expect(cardStyles.base.backgroundColor).toBe(colors.bgDefault);
      expect(cardStyles.base.borderColor).toBe(colors.borderDefault);
      expect(cardStyles.base.borderRadius).toBe('0.5rem');
    });

    it('exports selected card styles', () => {
      expect(cardStyles.selected).toBeDefined();
      expect(cardStyles.selected.borderColor).toBeDefined();
    });
  });

  describe('dialogStyles', () => {
    it('exports backdrop styles', () => {
      expect(dialogStyles.backdrop.position).toBe('fixed');
      expect(dialogStyles.backdrop.zIndex).toBe(50);
    });

    it('exports container styles', () => {
      expect(dialogStyles.container.display).toBe('flex');
      expect(dialogStyles.container.alignItems).toBe('center');
    });

    it('exports content styles', () => {
      expect(dialogStyles.content.borderRadius).toBe('0.5rem');
      expect(dialogStyles.content.maxWidth).toBe('32rem');
    });

    it('exports header, body, and footer styles', () => {
      expect(dialogStyles.header.padding).toBe('1rem');
      expect(dialogStyles.body.padding).toBe('1rem');
      expect(dialogStyles.footer.display).toBe('flex');
    });
  });

  describe('buttonColors', () => {
    it('exports success button colors', () => {
      expect(buttonColors.success.bg).toBeDefined();
      expect(buttonColors.success.text).toBeDefined();
    });

    it('exports danger button colors', () => {
      expect(buttonColors.danger.bg).toBeDefined();
      expect(buttonColors.danger.text).toBeDefined();
    });

    it('exports primary button colors', () => {
      expect(buttonColors.primary.bg).toBeDefined();
      expect(buttonColors.primary.text).toBeDefined();
    });
  });

  describe('badgeColors', () => {
    it('exports custom badge colors', () => {
      expect(badgeColors.custom.bg).toBeDefined();
      expect(badgeColors.custom.text).toBeDefined();
    });

    it('exports default badge colors', () => {
      expect(badgeColors.default.bg).toBeDefined();
      expect(badgeColors.default.text).toBeDefined();
    });
  });
});

