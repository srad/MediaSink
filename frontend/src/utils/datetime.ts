import moment from "moment";

/**
 * Returns a human-readable relative time string using moment.js.
 * Pass a timestamp (milliseconds since epoch) and it will return
 * the time that has passed since that date in a human-readable format.
 *
 * Examples:
 *   - "14 years ago"
 *   - "a month ago"
 *   - "2 days ago"
 *   - "an hour ago"
 *   - "23 minutes ago"
 *   - "a few seconds ago"
 *
 * @see https://momentjs.com/docs/#/displaying/fromnow/
 * @param  {number} Timestamp in milliseconds since epoch
 * @return {string} Human-readable relative time string
 */
export function fromNow(date: number): string {
  return moment(date).fromNow();
}

export function humanizeMs(ms: number): string {
  const tempTime = moment.duration(ms);
  return  tempTime.humanize()
}
