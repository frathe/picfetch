import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import test from 'node:test';
import {runInNewContext} from 'node:vm';

const page = readFileSync(new URL('./search_review.html', import.meta.url), 'utf8');
const script = page.match(/<script>([\s\S]*?)<\/script>/)[1];

// Execute the shipped download handler; only the browser boundary is synthetic.
function openReview(relevantIDs, failed = false) {
    const corpus = {
        version: 1,
        sources: Array.from({length: 33}, (_, i) => ({id: `s-${i}`, path: `${i}.jpg`})),
        queries: [{reference_id: 's-0', intent: 'content', relevant_ids: relevantIDs}],
    };
    const inputs = Array.from({length: 30}, (_, i) => ({
        value: `s-${i + 1}`,
        dataset: {rank: String(i), initial: String(relevantIDs?.includes(`s-${i + 1}`) ?? false)},
    }));
    const reviewed = {dataset: {initial: String(relevantIDs !== null)}};
    const label = {textContent: ''};
    const section = {
        dataset: {query: '0', intent: 'content'},
        querySelector(selector) {
            if (selector === '[data-reviewed]') return failed ? null : reviewed;
            if (selector === '[data-precision]') return failed ? null : label;
            throw new Error(`Unexpected section selector: ${selector}`);
        },
        querySelectorAll(selector) {
            if (selector === 'input[data-rank]') return inputs;
            if (selector === 'input[data-rank]:checked') return inputs.filter(input => input.checked);
            throw new Error(`Unexpected input selector: ${selector}`);
        },
    };
    let click, change, download;
    const document = {
        getElementById(id) {
            if (id === 'corpus') return {textContent: JSON.stringify(corpus)};
            if (id === 'review-status') return {textContent: ''};
            if (id === 'download') return {addEventListener: (event, callback) => {
                assert.equal(event, 'click');
                click = callback;
            }};
            throw new Error(`Unexpected element: ${id}`);
        },
        querySelectorAll(selector) {
            if (selector === '[data-query]') return [section];
            if (selector === 'input[data-initial]') return failed ? [] : [reviewed, ...inputs];
            throw new Error(`Unexpected document selector: ${selector}`);
        },
        addEventListener(event, callback) {
            assert.equal(event, 'change');
            change = callback;
        },
        createElement(tag) {
            assert.equal(tag, 'a');
            return {click() {}};
        },
    };
    runInNewContext(script, {
        document,
        Blob: class {
            constructor(parts) { download = parts.join(''); }
        },
        URL: {createObjectURL: () => 'blob:review', revokeObjectURL() {}},
        setTimeout: callback => callback(),
    });
    return {corpus, inputs, reviewed, label, change: () => change(), download: () => {
        click();
        return JSON.parse(download);
    }};
}

test('export preserves undisplayed labels while applying visible edits', () => {
    const review = openReview(['s-5', 's-31', 's-32']);
    review.inputs[4].checked = false;
    review.inputs[3].checked = true;
    review.change();
    const exported = review.download();
    assert.deepEqual(exported.queries[0].relevant_ids, ['s-31', 's-32', 's-4']);
    assert.deepEqual(exported.sources, review.corpus.sources);
    assert.equal(review.label.textContent, '0.100');
    assert.deepEqual(review.corpus.queries[0].relevant_ids, ['s-5', 's-31', 's-32']);

    review.inputs[3].checked = false;
    review.inputs[4].checked = true;
    assert.deepEqual(review.download().queries[0].relevant_ids, ['s-31', 's-32', 's-5']);
});

test('export distinguishes an unreviewed reference from reviewed zero matches', () => {
    const review = openReview(null);
    assert.equal(review.download().queries[0].relevant_ids, null);
    review.reviewed.checked = true;
    assert.deepEqual(review.download().queries[0].relevant_ids, []);
    review.inputs[0].checked = true;
    assert.deepEqual(review.download().queries[0].relevant_ids, ['s-1']);
    review.reviewed.checked = false;
    assert.equal(review.download().queries[0].relevant_ids, null);
});

test('export preserves labels when a failed reference has no review controls', () => {
    const review = openReview(['s-5', 's-31'], true);
    assert.deepEqual(review.download(), review.corpus);
});
