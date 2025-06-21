import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

// MLS (Major League Soccer) league id
const LEAGUE_IDS: Record<string, number> = {
	MLS: 253,
} as const;

export type OK<Data> = {
	ok: true;
	value: Data;
	error: null;
};
export function ok<Data>(data: Data): OK<Data> {
	return {
		ok: true,
		get value() {
			return data;
		},
		error: null,
	};
}

export type NotOK<Error> = { ok: false; error: Error };
export function err<E>(error: E): NotOK<E> {
	return {
		ok: false,
		error,
	};
}

export type Result<Data, Error = null> = OK<Data> | NotOK<Error>;

const headers = {
	"x-rapidapi-key": "91be9b12c36d01fd71847355d020c8d7",
	Accept: "application/json",
};

async function makeRequest<T>(
	path: string,
	params: Record<string, any>,
): Promise<Result<T, string>> {
	try {
		const response = await fetch(
			`https://v3.football.api-sports.io${path}?${new URLSearchParams(
				params,
			).toString()}`,
			{
				method: "GET",
				headers,
			},
		);
		if (!response.ok) {
			return err(`HTTP error! status: ${response.status}`);
		}
		return ok((await response.json()) as T);
	} catch (error) {
		console.error("Error making request:", error);
		return err(`${error}`);
	}
}

const server = new McpServer({
	name: "maestro",
	version: "1.0.0",
	capabilities: {
		resources: {},
		tools: {},
	},
});

type TeamsResponse = {
	parameters: {
		league: string;
		season: string;
	};
	errors: [];
	results: number;
	response: [
		{
			team: {
				id: number;
				name: string;
				code: string;
				country: string;
				founded: number;
				national: boolean;
				logo: string;
			};
		},
	];
};

server.tool(
	"get-teams",
	"Get a list of teams in a league",
	{
		league: z.enum(["MLS"]),
		season: z.number().describe("The season year (YYYY)"),
	},
	async ({ league, season }) => {
		const result = await makeRequest<TeamsResponse>("/teams", {
			league: LEAGUE_IDS[league],
			season,
		});
		if (!result.ok) {
			return {
				content: [
					{
						type: "text",
						text: `Failed to fetch teams: ${result.error}`,
					},
				],
			};
		}

		return {
			content: [
				{
					type: "text",
					text: JSON.stringify(result.value),
				},
			],
		};
	},
);

// server.tool(
// 	"get-leagues",
// 	"Get a list of leagues",
// 	{
// 		id: z.string().optional().describe("id of the league"),
// 		search: z.string().min(3).optional().describe("search term "),
// 		current: z
// 			.enum(["true", "false"])
// 			.optional()
// 			.describe("current state of the league"),
// 	},
// 	async (args) => {
// 		const result = await makeRequest<{
// 			response: { id: number; name: string }[];
// 		}>("/leagues");
// 		if (result.error) {
// 			throw new Error(result.error);
// 		}
// 		// todo: return the list as json content
// 		return { content: [] };
// 	},
// );

const transport = new StdioServerTransport();
await server.connect(transport);

console.error("MCP Soccer Statistics Server running");
