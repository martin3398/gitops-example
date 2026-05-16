type Params = {
  notice?: string;
  error?: string;
};

type Result = {
  message: string;
  toneClassName: "ok" | "err";
};

export function useStatusMessage({ notice, error }: Params): Result {
  if (error) {
    return { message: error, toneClassName: "err" };
  }

  return { message: notice ?? "", toneClassName: "ok" };
}
