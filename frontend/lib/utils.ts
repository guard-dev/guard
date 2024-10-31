import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const titleCase = (str: string) => {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
};


export const getTimeDifference = (
  unixTimestamp: number,
  full?: boolean,
): string => {
  const currentTime = Date.now();
  const differenceInMillis = currentTime - unixTimestamp; // Convert seconds to milliseconds

  const differenceInMinutes = Math.floor(differenceInMillis / (1000 * 60));
  const differenceInHours = Math.floor(differenceInMinutes / 60);
  const differenceInDays = Math.floor(differenceInHours / 24);

  if (differenceInDays >= 1) {
    if (full) {
      return `${differenceInDays} ${differenceInDays === 1 ? 'day' : 'days'}`;
    }
    return `${differenceInDays}d`;
  } else if (differenceInHours >= 1) {
    if (full) {
      return `${differenceInHours} ${differenceInHours === 1 ? 'hour' : 'hours'}`;
    }
    return `${differenceInHours}h`;
  } else {
    if (full) {
      return `${differenceInMinutes} ${differenceInMinutes === 1 ? 'minute' : 'minutes'}`;
    }
    return `${differenceInMinutes}m`;
  }
};

export const formatTimestamp = (unixTimestamp: number) => {
  const date = new Date(unixTimestamp);

  const optionsDate: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
  };
  const optionsTime: Intl.DateTimeFormatOptions = {
    hour: 'numeric',
    hour12: true,
  };

  const formattedDate = date.toLocaleDateString('en-US', optionsDate);
  const formattedTime = date
    .toLocaleTimeString('en-US', optionsTime)
    .toLowerCase()
    .replace(/\s/g, '');

  return `${formattedDate}, ${formattedTime}`;
};
