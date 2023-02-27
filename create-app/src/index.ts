import * as childProcess from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import prompts from 'prompts';
import validatePackageName from 'validate-npm-package-name';

async function getProjectPath() {
	const { id } = await prompts({
		type: 'text',
		name: 'id',
		message: 'What will be the package name for your app? (i.e. "my-app")',
		validate: (value) => {
			const { errors } = validatePackageName(value);
			if (errors) {
				return errors[0];
			}
			return true;
		},
	});
	const { projectRelPath } = await prompts({
		type: 'text',
		name: 'projectRelPath',
		message: 'Where should the app be created?',
		initial: id,
		validate: (projectRelPath) => {
			const projectPath = path.resolve(process.cwd(), projectRelPath);
			try {
				fs.statSync(projectPath);
			} catch (err) {
				return (err as { errno: string }).errno === 'ENOENT' ? true : err;
			}
			return 'The directory already exists';
		},
	});

	if (!id || !projectRelPath) {
		process.exit();
	}
	return { id, projectPath: path.resolve(process.cwd(), projectRelPath) };
}

const excitingEmojis = ['👋', '🎉', '🎊', '🎈', '🎁', '🎀', '🚀'];

const pickRand = <T>(arr: Array<T>): T =>
	arr[Math.floor(Math.random() * arr.length)];

const writeJson = async (path: string, obj: unknown) => {
	return fs.promises.writeFile(path, JSON.stringify(obj, null, '\t') + os.EOL);
};

function execExists(name) {
	try {
		childProcess.execSync(`which ${name}`);
		return true;
	} catch {
		return false;
	}
}

async function main() {
	const { id, projectPath } = await getProjectPath();
	const name = id.replace(/[^\w]([a-z])/g, (_, w) => w.toUpperCase());

	if (!id || !name) {
		return;
	}

	// Generate some codes
	await fs.promises.mkdir(path.resolve(projectPath, 'src'), {
		recursive: true,
	});
	await writeJson(path.resolve(projectPath, 'package.json'), {
		name: id,
		version: '0.0.0',
		license: 'MIT',
		scripts: {
			lint: 'rome check .',
			format: 'rome format --write .',
		},
		dependencies: {
			react: '^18',
			'@robinplatform/toolkit': '^0',
		},
		devDependencies: {
			rome: '^11',
		},
	});
	await writeJson(path.resolve(projectPath, 'robin.app.json'), {
		id,
		name,
		page: './src/app.tsx',
		pageIcon: pickRand(excitingEmojis),
	});
	await writeJson(path.resolve(projectPath, 'rome.json'), {
		$schema: './node_modules/rome/configuration_schema.json',
		linter: {
			enabled: true,
			rules: {
				recommended: true,
			},
		},
		formatter: {
			enabled: true,
			indentStyle: 'tab',
		},
	});
	await fs.promises.writeFile(
		path.resolve(projectPath, '.gitignore'),
		['node_modules', '*.js', '*.log'].join(os.EOL) + os.EOL,
	);
	await fs.promises.writeFile(
		path.resolve(projectPath, 'src', 'app.tsx'),
		[
			`import React from "react";`,
			`import { renderApp } from "@robinplatform/toolkit/react";`,
			`import { useRpcQuery } from "@robinplatform/toolkit/react/rpc";`,
			`import { getOsInfo } from "./app.server";`,
			``,
			`const App = () => {`,
			`\tconst { data: osInfo } = useRpcQuery(getOsInfo, {});`,
			`\tif (!osInfo) return <p>Loading...</p>;`,
			`\treturn <p>You're on {osInfo?.platform}!</p>;`,
			`};`,
			``,
			`renderApp(<App />);`,
			``,
		].join(os.EOL),
	);
	await fs.promises.writeFile(
		path.resolve(projectPath, 'src', 'app.server.ts'),
		[
			`import * as os from "os";`,
			``,
			`export async function getOsInfo() {`,
			`\treturn {`,
			`\t\tplatform: os.platform(),`,
			`\t};`,
			`}`,
			``,
		].join(os.EOL),
	);

	await childProcess.execSync(
		`${execExists('yarn') ? 'yarn' : 'npm'} install`,
		{
			cwd: projectPath,
			stdio: 'inherit',
		},
	);
}

main();
