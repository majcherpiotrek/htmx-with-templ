import type { ZodType } from "zod";


export const getServerProps = <T>(propsId: string, schema: ZodType<T>): T => {
  try {
    const rawJson = JSON.parse(document.getElementById(propsId)?.textContent ?? "");

    return schema.parse(rawJson)
  } catch (error: unknown) {
    console.error(`Failed to get json for propsId=${propsId}`);
    console.error(error);

    throw error;
  }
}

