import {
  CompletionId, CompletionItem, CompletionItems, CompletionTitle,
} from './completions';

export type ItemsMap = Map<CompletionId, { title: CompletionTitle; index: number; type: string }>;

// Finds the next completion item in the list, given the id of the completion that is currently active.
export const findNextItem = (activeItem: CompletionId, itemsMap: ItemsMap, completions: CompletionItems, direction = 1):
CompletionId => {
  if (completions.length === 0) {
    return '';
  }

  let index = -1;
  if (activeItem !== '') {
    const item = itemsMap.get(activeItem);
    ({ index } = item);
  }
  const { length } = completions;
  for (let i = 1; i <= length; i++) {
    const nextIndex = (index + (i * direction) + length) % length;
    const next = completions[nextIndex] as CompletionItem;
    if (next.title && next.id) {
      return next.id;
    }
  }
  return activeItem;
};

// Represents a key/value field in the autocomplete command.
export interface TabStop {
  Index: number;
  Label?: string;
  Value?: string;
  CursorPosition: number;
}

// getDisplayStringFromTabStops parses the tabstops to get the display text
// that is shown to the user in the input box.
export const getDisplayStringFromTabStops = (tabStops) => {
  let str = '';
  tabStops.forEach((ts, index) => {
    if (ts.Label !== undefined) {
      str += `${ts.Label}:`;
    }
    if (ts.Value !== undefined) {
      str += ts.Value;
    }

    if (index !== tabStops.length - 1) {
      str += ' ';
    }
  });

  return str;
};

// Tracks information about the tabstop, such as what the formatted input should look like and the boundaries of
// each tabstop in the display string.
export class TabStopParser {
  private tabStops;

  private tabBoundaries;

  private input;

  private initialCursor;

  constructor(tabStops: Array<TabStop>) {
    this.parseTabStopInfo(tabStops);
  }

  // parseTabStopInfo parses the tabstops into useful display information.
  private parseTabStopInfo = (tabStops: Array<TabStop>) => {
    let cursorPos = 0;
    let currentPos = 0;
    this.tabStops = tabStops;
    this.input = [];
    this.tabBoundaries = [];
    tabStops.forEach((ts, index) => {
      if (ts.Label !== undefined) {
        this.input.push({ type: 'key', value: `${ts.Label}:` });
        currentPos += ts.Label.length + 1;
      }
      if (ts.CursorPosition !== -1) {
        cursorPos = currentPos + ts.CursorPosition;
      }

      const valueStartIndex = currentPos;
      if (ts.Value !== undefined) {
        this.input.push({ type: 'value', value: ts.Value });
        currentPos += ts.Value.length;
      }

      this.tabBoundaries.push([valueStartIndex, currentPos + 1]);

      if (index !== tabStops.length - 1) {
        this.input.push({ type: 'value', value: ' ' });
        currentPos += 1;
      }
    });

    this.initialCursor = cursorPos;
  };

  public getInitialCursor = () => (this.initialCursor);

  public getTabBoundaries = () => (this.tabBoundaries);

  public getInput = () => (this.input);

  // Find the tabstop that the cursor is currently in.
  public getActiveTab = (cursorPos: number): number => {
    let tabIdx = -1;
    this.tabBoundaries.forEach((boundary, index) => {
      if (tabIdx === -1 && cursorPos < boundary[1]) {
        tabIdx = index;
      }
    });

    return tabIdx;
  };

  // handleCompletionSelection returns what the new display string should look like
  // if the completion was made for the given activeTab. It does not actually
  // mutate the contents of the current tabstops.
  public handleCompletionSelection = (cursorPos: number, completion): [string, number] => {
    const activeTab = this.getActiveTab(cursorPos);

    const newTStops = [];
    let newCursorPos = -1;

    this.tabStops.forEach((ts, i) => {
      if (i === activeTab) {
        const newTS = { Index: ts.Index } as TabStop;

        if (ts.Label === undefined) {
          newTS.Label = completion.type;
        } else {
          newTS.Label = ts.Label;
        }

        newTS.Value = completion.title;

        if (activeTab > 0) {
          newCursorPos = this.tabBoundaries[activeTab - 1][1];
        }
        newCursorPos += newTS.Label.length + newTS.Value.length + 1;
        newTStops.push(newTS);
      } else {
        newTStops.push({
          Index: ts.Index,
          Label: ts.Label,
          Value: ts.Value,
          CursorPosition: ts.Cursor,
        });
      }
    });

    return [getDisplayStringFromTabStops(newTStops), newCursorPos];
  };

  // handleBackspace returns what the new display string should look like
  // if a backspace was made. It does not actually mutate the contents of the
  // current tabstop.
  public handleBackspace = (cursorPos: number): [string, number] => {
    const activeTab = this.getActiveTab(cursorPos);

    if (this.tabBoundaries[activeTab][0] !== cursorPos) { // User is deleting within the current tabstop.
      const str = getDisplayStringFromTabStops(this.tabStops);
      const newStr = str.substring(0, cursorPos - 1) + str.substring(cursorPos);
      return [newStr, cursorPos - 1];
    }
    // Delete the whole tabstop.
    const str = getDisplayStringFromTabStops(this.tabStops.slice(0, activeTab).concat(
      this.tabStops.slice(activeTab + 1),
    ));
    if (activeTab === this.tabStops.length) { //
      return [str, this.tabBoundaries[activeTab - 1][1] - 2];
    }

    if (activeTab === 0) {
      return [str, 0];
    }
    return [str, this.tabBoundaries[activeTab - 1][1] - 1];
  };

  // handleBackspace returns what the new display string should look like
  // if the change was made in the current position.
  public handleChange = (input: string, cursorPos: number): Array<TabStop> => {
    const char = input[cursorPos - 1]; // Get character typed by user.
    const activeTab = this.getActiveTab(cursorPos - 1);
    const newTStops = [];

    this.tabStops.forEach((ts, i) => {
      if (i === activeTab) {
        let value = ts.Value;
        let tsCursor = -1;
        const pos = (cursorPos - 1) - this.tabBoundaries[i][0];
        if (value != null) {
          value = ts.Value.substring(0, pos) + char + ts.Value.substring(pos);
          tsCursor = pos + 1;
        } else {
          value = char;
          tsCursor = 1;
        }
        newTStops.push({
          Index: ts.Index, Label: ts.Label, CursorPosition: tsCursor, Value: value,
        });
      } else {
        newTStops.push({
          Index: ts.Index,
          Label: ts.Label,
          Value: ts.Value,
          CursorPosition: ts.CursorPosition,
        });
      }
    });

    return newTStops;
  };
}
