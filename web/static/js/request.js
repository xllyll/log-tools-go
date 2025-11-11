/**
 * 请求封装
 * @param url
 * @param method
 * @param data
 * @param header
 * @returns {Promise<unknown>}
 */
const baseRequest = (url, method, data, header) => {
    return new Promise((resolve, reject) => {
        let options = {
            method: method
        }
        if (data !== null && data !== undefined) {
            options['body'] = JSON.stringify(data)
        }
        if (header === null || header === undefined) {
            options['headers'] = {
                'Content-Type': 'application/json'
            }
        }else{
            options['headers'] = header;
        }
        fetch(url, options).then(res => res.json()).then(data => {
            resolve(data);
        }).catch((err) => {
            reject(err);
        })
    })
}